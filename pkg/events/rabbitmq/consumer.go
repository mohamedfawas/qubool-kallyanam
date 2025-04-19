package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ConsumeOptions contains options for consuming messages
type ConsumeOptions struct {
	QueueName  string
	ConsumerID string
	AutoAck    bool
	Exclusive  bool
}

// MessageHandler is a function type that processes a delivery
type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

// Consumer handles consuming messages from RabbitMQ
type Consumer struct {
	client  *Client
	logger  *zap.Logger
	options ConsumeOptions
	handler MessageHandler
	msgChan <-chan amqp.Delivery
	stopCh  chan struct{}
}

// NewConsumer creates a new consumer
func NewConsumer(client *Client, options ConsumeOptions, logger *zap.Logger) *Consumer {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Consumer{
		client:  client,
		logger:  logger,
		options: options,
		stopCh:  make(chan struct{}),
	}
}

// Start begins consuming messages and passes them to the handler
func (c *Consumer) Start(ctx context.Context, handler MessageHandler) error {
	c.handler = handler

	var err error
	c.msgChan, err = c.client.Channel().Consume(
		c.options.QueueName,
		c.options.ConsumerID,
		c.options.AutoAck,
		c.options.Exclusive,
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.logger.Info("Consumer started",
		zap.String("queue", c.options.QueueName),
		zap.String("consumer_id", c.options.ConsumerID),
	)

	go c.consume(ctx)

	return nil
}

// consume processes incoming messages
func (c *Consumer) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context canceled, stopping consumer")
			return
		case <-c.stopCh:
			c.logger.Info("Stop signal received, stopping consumer")
			return
		case delivery, ok := <-c.msgChan:
			if !ok {
				c.logger.Warn("Delivery channel closed")
				return
			}

			c.processDelivery(ctx, delivery)
		}
	}
}

// processDelivery handles a single delivery
func (c *Consumer) processDelivery(ctx context.Context, delivery amqp.Delivery) {
	c.logger.Debug("Received message",
		zap.String("exchange", delivery.Exchange),
		zap.String("routing_key", delivery.RoutingKey),
		zap.String("content_type", delivery.ContentType),
		zap.Int("body_size", len(delivery.Body)),
	)

	err := c.handler(ctx, delivery)
	if err != nil {
		c.logger.Error("Error processing message",
			zap.Error(err),
			zap.String("message_id", delivery.MessageId),
		)

		// Only reject if we haven't auto-acked
		if !c.options.AutoAck {
			if err := delivery.Reject(false); err != nil {
				c.logger.Error("Failed to reject message", zap.Error(err))
			}
		}
		return
	}

	// Acknowledge message if not using auto-ack
	if !c.options.AutoAck {
		if err := delivery.Ack(false); err != nil {
			c.logger.Error("Failed to acknowledge message", zap.Error(err))
		}
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	close(c.stopCh)
}

// UnmarshalJSON unmarshals a delivery body to a struct
func UnmarshalJSON(delivery amqp.Delivery, v interface{}) error {
	if delivery.ContentType != "application/json" {
		return fmt.Errorf("unexpected content type: %s", delivery.ContentType)
	}

	return json.Unmarshal(delivery.Body, v)
}
