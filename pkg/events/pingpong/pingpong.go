package pingpong

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/events/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	// ExchangeName for the ping-pong service
	ExchangeName = "ping.pong"
	// PingRoutingKey for ping messages
	PingRoutingKey = "ping"
	// PongRoutingKey for pong messages
	PongRoutingKey = "pong"
)

// Message represents a ping or pong message
type Message struct {
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// PingService handles sending pings and receiving pongs
type PingService struct {
	publisher   *rabbitmq.Publisher
	consumer    *rabbitmq.Consumer
	client      *rabbitmq.Client
	logger      *zap.Logger
	serviceName string
}

// NewPingService creates a new ping service
func NewPingService(
	client *rabbitmq.Client,
	serviceName string,
	logger *zap.Logger,
) (*PingService, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create the exchange
	if err := client.DeclareExchange(ExchangeName, "direct", true, false); err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Create the ping queue
	pingQueue, err := client.DeclareQueue(
		fmt.Sprintf("ping.%s", serviceName),
		true,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare ping queue: %w", err)
	}

	// Bind the queue to the exchange
	if err := client.BindQueue(pingQueue.Name, PingRoutingKey, ExchangeName); err != nil {
		return nil, fmt.Errorf("failed to bind ping queue: %w", err)
	}

	// Create the pong queue
	pongQueue, err := client.DeclareQueue(
		fmt.Sprintf("pong.%s", serviceName),
		true,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare pong queue: %w", err)
	}

	// Bind the queue to the exchange
	if err := client.BindQueue(pongQueue.Name, PongRoutingKey, ExchangeName); err != nil {
		return nil, fmt.Errorf("failed to bind pong queue: %w", err)
	}

	publisher := rabbitmq.NewPublisher(client, logger)

	consumer := rabbitmq.NewConsumer(client, rabbitmq.ConsumeOptions{
		QueueName:  pingQueue.Name,
		ConsumerID: fmt.Sprintf("ping-consumer-%s", serviceName),
		AutoAck:    false,
		Exclusive:  false,
	}, logger)

	return &PingService{
		publisher:   publisher,
		consumer:    consumer,
		client:      client,
		logger:      logger,
		serviceName: serviceName,
	}, nil
}

// Start starts the ping service
func (p *PingService) Start(ctx context.Context) error {
	// Start consuming ping messages
	return p.consumer.Start(ctx, p.handlePing)
}

// handlePing processes ping messages and replies with pong
func (p *PingService) handlePing(ctx context.Context, delivery amqp.Delivery) error {
	var msg Message
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal ping message: %w", err)
	}

	p.logger.Info("Received ping",
		zap.String("from", msg.Sender),
		zap.String("message", msg.Message),
	)

	// Reply with a pong
	pongMsg := Message{
		Sender:    p.serviceName,
		Message:   fmt.Sprintf("Pong in response to: %s", msg.Message),
		Timestamp: time.Now(),
	}

	opts := rabbitmq.DefaultPublishOptions()
	opts.Exchange = ExchangeName
	opts.RoutingKey = PongRoutingKey

	if err := p.publisher.PublishJSON(ctx, pongMsg, opts); err != nil {
		return fmt.Errorf("failed to publish pong: %w", err)
	}

	return nil
}

// SendPing sends a ping message
func (p *PingService) SendPing(ctx context.Context, message string) error {
	msg := Message{
		Sender:    p.serviceName,
		Message:   message,
		Timestamp: time.Now(),
	}

	opts := rabbitmq.DefaultPublishOptions()
	opts.Exchange = ExchangeName
	opts.RoutingKey = PingRoutingKey

	if err := p.publisher.PublishJSON(ctx, msg, opts); err != nil {
		return fmt.Errorf("failed to publish ping: %w", err)
	}

	p.logger.Info("Sent ping message",
		zap.String("message", message),
	)

	return nil
}

// Stop stops the ping service
func (p *PingService) Stop() {
	p.consumer.Stop()
}
