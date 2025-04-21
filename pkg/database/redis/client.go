package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"qubool-kallyanam/pkg/database"
	"qubool-kallyanam/pkg/logging"
)

// Config extends the common database config with Redis specific options
type Config struct {
	database.Config
	DB              int           `yaml:"db"`
	MinIdleConns    int           `yaml:"min_idle_conns"`
	PoolSize        int           `yaml:"pool_size"`
	PoolTimeout     time.Duration `yaml:"pool_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	MaxRetries      int           `yaml:"max_retries"`
	MinRetryBackoff time.Duration `yaml:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `yaml:"max_retry_backoff"`
}

// Client is a Redis client
type Client struct {
	client *redis.Client
	config Config
	logger logging.Logger
}

// NewClient creates a new Redis client
func NewClient(config Config, logger logging.Logger) *Client {
	if logger == nil {
		logger = logging.Get().Named("redis")
	}

	return &Client{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to Redis
func (c *Client) Connect(ctx context.Context) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		Password: c.config.Password,
		DB:       c.config.DB,
	}

	// Configure connection pool
	if c.config.PoolSize > 0 {
		opts.PoolSize = c.config.PoolSize
	}
	if c.config.MinIdleConns > 0 {
		opts.MinIdleConns = c.config.MinIdleConns
	}
	if c.config.PoolTimeout > 0 {
		opts.PoolTimeout = c.config.PoolTimeout
	}

	// Configure timeouts
	if c.config.ConnTimeout > 0 {
		opts.DialTimeout = c.config.ConnTimeout
	}
	if c.config.ReadTimeout > 0 {
		opts.ReadTimeout = c.config.ReadTimeout
	}
	if c.config.WriteTimeout > 0 {
		opts.WriteTimeout = c.config.WriteTimeout
	}

	// Configure retries
	if c.config.MaxRetries > 0 {
		opts.MaxRetries = c.config.MaxRetries
	}
	if c.config.MinRetryBackoff > 0 {
		opts.MinRetryBackoff = c.config.MinRetryBackoff
	}
	if c.config.MaxRetryBackoff > 0 {
		opts.MaxRetryBackoff = c.config.MaxRetryBackoff
	}

	// Create client
	client := redis.NewClient(opts)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	c.client = client
	c.logger.Info("Connected to Redis",
		logging.String("host", c.config.Host),
		logging.Int("port", c.config.Port),
		logging.Int("db", c.config.DB),
	)

	return nil
}

// Close closes the Redis connection
func (c *Client) Close(ctx context.Context) error {
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			return fmt.Errorf("failed to close redis connection: %w", err)
		}
		c.logger.Info("Closed Redis connection")
	}
	return nil
}

// Ping verifies the connection to Redis
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("redis client not connected")
	}
	return c.client.Ping(ctx).Err()
}

// Stats returns connection pool statistics
func (c *Client) Stats() interface{} {
	if c.client == nil {
		return nil
	}
	return c.client.PoolStats()
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}
