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
	reconnectDelay = 5 * time.Second
	publishTimeout = 5 * time.Second
)

// MessageHandler is a function that processes received messages
type MessageHandler func(message []byte) error

// Client represents a RabbitMQ client
type Client struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	dsn          string
	exchangeName string
	isConnected  bool
	handlers     map[string]MessageHandler
}

// NewClient creates a new RabbitMQ client
func NewClient(dsn string, exchangeName string) (*Client, error) {
	client := &Client{
		dsn:          dsn,
		exchangeName: exchangeName,
		handlers:     make(map[string]MessageHandler),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	go client.reconnectLoop()

	return client, nil
}

// connect establishes connection to RabbitMQ
func (c *Client) connect() error {
	conn, err := amqp.Dial(c.dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the exchange if it doesn't exist
	err = ch.ExchangeDeclare(
		c.exchangeName, // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	c.conn = conn
	c.channel = ch
	c.isConnected = true

	// Set up connection close notifier
	go func() {
		<-c.conn.NotifyClose(make(chan *amqp.Error))
		c.isConnected = false
	}()

	return nil
}

// reconnectLoop attempts to reconnect when connection is lost
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

				// Resubscribe to all event types
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

// Publish sends a message to specified event type
func (c *Client) Publish(eventType string, data interface{}) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	return c.channel.PublishWithContext(
		ctx,
		c.exchangeName, // exchange
		eventType,      // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		},
	)
}

// Subscribe registers a handler for a specific event type
func (c *Client) Subscribe(eventType string, handler MessageHandler) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	// Store the handler
	c.handlers[eventType] = handler

	return c.setupSubscription(eventType)
}

// setupSubscription creates a queue and binding for an event type
func (c *Client) setupSubscription(eventType string) error {
	// Create a queue with a unique name for the service
	queue, err := c.channel.QueueDeclare(
		"",    // name (empty = auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	// Bind the queue to the exchange
	err = c.channel.QueueBind(
		queue.Name,     // queue name
		eventType,      // routing key
		c.exchangeName, // exchange
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming messages
	msgs, err := c.channel.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	// Process messages
	go func() {
		for msg := range msgs {
			handler, ok := c.handlers[eventType]
			if !ok {
				msg.Ack(false)
				continue
			}

			err := handler(msg.Body)
			if err != nil {
				log.Printf("Error handling message: %v", err)
				msg.Nack(false, true) // negative acknowledge, requeue
			} else {
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
