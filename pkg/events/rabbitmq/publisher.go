package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// PublishOptions contains options for publishing messages
type PublishOptions struct {
	Exchange     string
	RoutingKey   string
	Mandatory    bool
	Immediate    bool
	ContentType  string
	DeliveryMode uint8 // 1 = non-persistent, 2 = persistent
	Priority     uint8
	Expiration   string
	MessageID    string
}

// DefaultPublishOptions returns default publish options
func DefaultPublishOptions() PublishOptions {
	return PublishOptions{
		Exchange:     "",
		RoutingKey:   "",
		Mandatory:    false,
		Immediate:    false,
		ContentType:  "application/json",
		DeliveryMode: 2, // persistent
		Priority:     0,
		Expiration:   "",
		MessageID:    "",
	}
}

// Publisher handles publishing messages to RabbitMQ
type Publisher struct {
	client *Client
	logger *zap.Logger
}

// NewPublisher creates a new publisher
func NewPublisher(client *Client, logger *zap.Logger) *Publisher {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Publisher{
		client: client,
		logger: logger,
	}
}

// Publish publishes a message with the given options
func (p *Publisher) Publish(ctx context.Context, body []byte, options PublishOptions) error {
	// Create publishing
	msg := amqp.Publishing{
		ContentType:  options.ContentType,
		DeliveryMode: options.DeliveryMode,
		Priority:     options.Priority,
		MessageId:    options.MessageID,
		Timestamp:    time.Now(),
		Body:         body,
	}

	if options.Expiration != "" {
		msg.Expiration = options.Expiration
	}

	// Publish the message
	err := p.client.Channel().PublishWithContext(
		ctx,
		options.Exchange,
		options.RoutingKey,
		options.Mandatory,
		options.Immediate,
		msg,
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("Message published",
		zap.String("exchange", options.Exchange),
		zap.String("routing_key", options.RoutingKey),
		zap.Int("body_size", len(body)),
	)

	return nil
}

// PublishJSON publishes a JSON-serialized message
func (p *Publisher) PublishJSON(ctx context.Context, data interface{}, options PublishOptions) error {
	// Set content type to JSON
	options.ContentType = "application/json"

	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	return p.Publish(ctx, jsonData, options)
}
