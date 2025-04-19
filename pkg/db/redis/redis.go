package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	MaxConns int
	MinIdle  int
	Timeout  time.Duration
}

// Client represents a Redis client
type Client struct {
	client *redis.Client
	logger *zap.Logger
}

// NewClient creates a new Redis client
func NewClient(ctx context.Context, cfg Config, serviceName string, logger *zap.Logger) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.MaxConns,
		MinIdleConns: cfg.MinIdle,
		DialTimeout:  cfg.Timeout,
	})

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if _, err := client.Ping(pingCtx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Connected to Redis",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("service", serviceName))

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping checks if the redis connection is alive
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx).Result()
	return err
}

// Client returns the underlying redis client
func (c *Client) Client() *redis.Client {
	return c.client
}
