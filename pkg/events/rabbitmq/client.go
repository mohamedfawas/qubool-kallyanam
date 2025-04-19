package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Config holds RabbitMQ connection configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	VHost    string
	// Connection settings
	Reconnect      bool
	ReconnectDelay time.Duration
}

// Client represents a RabbitMQ client
type Client struct {
	config     Config
	logger     *zap.Logger
	connection *amqp.Connection
	channel    *amqp.Channel
	// Notification channels
	connClosed chan *amqp.Error
	chanClosed chan *amqp.Error
}

// NewClient creates a new RabbitMQ client
func NewClient(config Config, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Client{
		config:     config,
		logger:     logger,
		connClosed: make(chan *amqp.Error),
		chanClosed: make(chan *amqp.Error),
	}
}

// Connect establishes a connection to RabbitMQ
func (c *Client) Connect() error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		c.config.Username, c.config.Password, c.config.Host, c.config.Port, c.config.VHost)

	var err error
	c.connection, err = amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.connClosed = c.connection.NotifyClose(make(chan *amqp.Error, 1))

	c.channel, err = c.connection.Channel()
	if err != nil {
		c.connection.Close()
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	c.chanClosed = c.channel.NotifyClose(make(chan *amqp.Error, 1))

	c.logger.Info("Connected to RabbitMQ", zap.String("host", c.config.Host), zap.Int("port", c.config.Port))

	// Start reconnection handler if enabled
	if c.config.Reconnect {
		go c.handleReconnect()
	}

	return nil
}

// handleReconnect attempts to reconnect when connection is lost
func (c *Client) handleReconnect() {
	for {
		select {
		case err := <-c.connClosed:
			if err != nil {
				c.logger.Error("Connection closed", zap.Error(err))
				time.Sleep(c.config.ReconnectDelay)
				c.reconnect()
			}
		case err := <-c.chanClosed:
			if err != nil {
				c.logger.Error("Channel closed", zap.Error(err))
				time.Sleep(c.config.ReconnectDelay)
				c.reconnectChannel()
			}
		}
	}
}

// reconnect attempts to reestablish the connection
func (c *Client) reconnect() {
	for {
		c.logger.Info("Attempting to reconnect to RabbitMQ")
		err := c.Connect()
		if err != nil {
			c.logger.Error("Failed to reconnect", zap.Error(err))
			time.Sleep(c.config.ReconnectDelay)
			continue
		}
		return
	}
}

// reconnectChannel attempts to reestablish the channel
func (c *Client) reconnectChannel() {
	for {
		c.logger.Info("Attempting to reopen channel")
		var err error
		c.channel, err = c.connection.Channel()
		if err != nil {
			c.logger.Error("Failed to reopen channel", zap.Error(err))
			time.Sleep(c.config.ReconnectDelay)
			continue
		}
		c.chanClosed = c.channel.NotifyClose(make(chan *amqp.Error, 1))
		return
	}
}

// Close closes the connection and channel
func (c *Client) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}

	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	c.logger.Info("RabbitMQ connection closed")
	return nil
}

// Channel returns the AMQP channel
func (c *Client) Channel() *amqp.Channel {
	return c.channel
}

// DeclareExchange declares an exchange
func (c *Client) DeclareExchange(name, kind string, durable, autoDelete bool) error {
	err := c.channel.ExchangeDeclare(
		name,
		kind,
		durable,
		autoDelete,
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	return nil
}

// DeclareQueue declares a queue
func (c *Client) DeclareQueue(name string, durable, autoDelete bool) (amqp.Queue, error) {
	return c.channel.QueueDeclare(
		name,
		durable,
		autoDelete,
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

// BindQueue binds a queue to an exchange
func (c *Client) BindQueue(queueName, key, exchangeName string) error {
	err := c.channel.QueueBind(
		queueName,
		key,
		exchangeName,
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}
	return nil
}
