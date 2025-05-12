// pkg/messaging/rabbitmq/rabbitmq.go
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// If connection to RabbitMQ is lost, wait this long before retrying.
	reconnectDelay = 5 * time.Second
	// When publishing a message, don’t wait longer than this.
	publishTimeout = 5 * time.Second
)

// MessageHandler is a callback function type that processes the raw message bytes.
// It returns an error if processing failed.
type MessageHandler func(message []byte) error

// Client holds details about our RabbitMQ connection, channel, and subscriptions.
type Client struct {
	conn         *amqp.Connection          // underlying network connection
	channel      *amqp.Channel             // channel multiplexed over conn
	dsn          string                    // Data Source Name (e.g. "amqp://guest:guest@localhost:5672/")
	exchangeName string                    // logical exchange (like a topic bus)
	isConnected  bool                      // tracks if conn+channel are live
	handlers     map[string]MessageHandler // map of routingKey -> handler function
}

// NewClient creates a Client, connects immediately, and starts auto-reconnect in background.
// dsn: RabbitMQ URL, e.g. "amqp://guest:guest@localhost:5672/"
// exchangeName: name of exchange to publish/subscribe on, e.g. "events"
func NewClient(dsn string, exchangeName string) (*Client, error) {
	client := &Client{
		dsn:          dsn,
		exchangeName: exchangeName,
		handlers:     make(map[string]MessageHandler),
	}

	// Try to establish initial connection
	if err := client.connect(); err != nil {
		return nil, err
	}

	// In a new goroutine, monitor connection and reconnect if lost
	go client.reconnectLoop()

	return client, nil
}

// connect dials RabbitMQ, opens a channel, and declares the exchange
func (c *Client) connect() error {
	// 1. Dial connects over TCP and AMQP handshake
	conn, err := amqp.Dial(c.dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// 2. Open a channel on the connection
	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // cleanup on error
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// 3. Declare an exchange of type "topic"
	//    - durable: survives broker restart
	//    - topic: allows routing keys with patterns
	err = ch.ExchangeDeclare(
		c.exchangeName, // name, e.g. "events"
		"topic",        // type
		true,           // durable
		false,          // auto-deleted when no queues
		false,          // internal (no)
		false,          // no-wait (wait for confirmation)
		nil,            // additional args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	// Save successful connection + channel
	c.conn = conn
	c.channel = ch
	c.isConnected = true

	// Start a watcher: when connection closes, mark isConnected=false
	go func() {
		// NotifyClose returns a channel that gets an *amqp.Error when closed
		<-c.conn.NotifyClose(make(chan *amqp.Error))
		c.isConnected = false
	}()

	return nil
}

// reconnectLoop checks connectivity every reconnectDelay and attempts reconnects
func (c *Client) reconnectLoop() {
	for {
		if !c.isConnected {
			log.Println("Attempting to reconnect to RabbitMQ...")
			for {
				if err := c.connect(); err != nil {
					log.Printf("Failed to reconnect: %v. Retrying in %s", err, reconnectDelay)
					time.Sleep(reconnectDelay)
					continue
				}
				log.Println("Reconnected to RabbitMQ")

				// After reconnect, re-establish subscriptions
				for eventType := range c.handlers {
					if err := c.setupSubscription(eventType); err != nil {
						log.Printf("Failed to resubscribe to %s: %v", eventType, err)
					}
				}
				break
			}
		}
		time.Sleep(reconnectDelay)
	}
}

// Publish sends a message to a specific routing key (eventType)
// data can be any serializable Go value (struct, map, etc.)
// Example: client.Publish("user.created", User{ID:123, Name:"Alice"})
func (c *Client) Publish(eventType string, data interface{}) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	// Convert data to JSON bytes
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create a context with timeout to avoid blocking forever
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	// Publish with context for safety
	return c.channel.PublishWithContext(
		ctx,
		c.exchangeName, // exchange
		eventType,      // routing key (topic)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
}

// Subscribe registers a handler function for messages on eventType
// Example: client.Subscribe("order.created", func(msg []byte){ ... })
func (c *Client) Subscribe(eventType string, handler MessageHandler) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	// Save handler in map
	c.handlers[eventType] = handler

	// Actually set up the queue, binding, and consumer
	return c.setupSubscription(eventType)
}

// setupSubscription creates a temporary private queue and binds it to the exchange key
func (c *Client) setupSubscription(eventType string) error {
	// 1. Declare a queue that the broker names (empty name)
	queue, err := c.channel.QueueDeclare(
		"",    // name: empty for auto-generated unique name
		false, // durable: not persisted
		true,  // delete when unused
		true,  // exclusive: only this connection can use it
		false, // no-wait: wait for server
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	// 2. Bind our queue to the exchange with the routing key eventType
	//    This means messages published with that routing key go into our queue
	err = c.channel.QueueBind(
		queue.Name,     // queue name from previous step
		eventType,      // routing key, e.g. "user.updated"
		c.exchangeName, // exchange name
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// 3. Start consuming messages off this queue
	msgs, err := c.channel.Consume(
		queue.Name, // queue
		"",         // consumer tag (empty auto-generates)
		false,      // auto-ack: false means we manually Ack/Nack
		false,      // exclusive: no other consumers
		false,      // no-local: allow own publishes
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	// 4. Process messages in a goroutine so consuming is asynchronous
	go func() {
		for msg := range msgs {
			// Retrieve our handler for this eventType
			handler, ok := c.handlers[eventType]
			if !ok {
				// If no handler found, just acknowledge and skip
				msg.Ack(false)
				continue
			}

			// Call handler with the raw byte payload
			err := handler(msg.Body)
			if err != nil {
				log.Printf("Error handling message: %v", err)
				// Negative ack: tell RabbitMQ to requeue for retry
				msg.Nack(false, true) // negative acknowledge, requeue
			} else {
				// Acknowledge successful processing
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// Close shuts down the channel and the connection cleanly
func (c *Client) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
