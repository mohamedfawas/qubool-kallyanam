package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
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
	// Set default values if not specified
	if config.PoolSize <= 0 {
		config.PoolSize = 10 // Default pool size
	}
	if config.MinIdleConns <= 0 {
		config.MinIdleConns = 5 // Default min idle connections
	}
	if config.ConnTimeout <= 0 {
		config.ConnTimeout = 5 * time.Second // Default connection timeout
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 3 * time.Second // Default read timeout
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 3 * time.Second // Default write timeout
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 3 // Default max retries
	}
	if config.Port <= 0 {
		config.Port = 6379 // Default Redis port
	}

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
	// Check for environment variables first - industry standard approach
	host := c.config.Host
	if envHost := os.Getenv("REDIS_HOST"); envHost != "" {
		host = envHost
		c.logger.Debug("Using REDIS_HOST from env", logging.String("host", host))
	}

	port := c.config.Port
	if envPort := os.Getenv("REDIS_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
			c.logger.Debug("Using REDIS_PORT from env", logging.Int("port", port))
		}
	}

	password := c.config.Password
	if envPass := os.Getenv("REDIS_PASSWORD"); envPass != "" {
		password = envPass
		c.logger.Debug("Using REDIS_PASSWORD from env")
	}

	db := c.config.DB
	if envDB := os.Getenv("REDIS_DB"); envDB != "" {
		if d, err := strconv.Atoi(envDB); err == nil {
			db = d
			c.logger.Debug("Using REDIS_DB from env", logging.Int("db", db))
		}
	}

	// Build Redis options
	opt := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,
		DialTimeout:  c.config.ConnTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
	}

	// Create client
	c.client = redis.NewClient(opt)

	c.logger.Info("Connected to Redis",
		logging.String("host", host),
		logging.Int("port", port),
		logging.Int("db", db),
	)

	// Test connection
	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
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

// ExecutePipeline executes commands in a Redis pipeline
func (c *Client) ExecutePipeline(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	if c.client == nil {
		return nil, fmt.Errorf("redis client not connected")
	}

	pipe := c.client.Pipeline()
	if err := fn(pipe); err != nil {
		return nil, err
	}

	return pipe.Exec(ctx)
}

// ExecuteTransaction executes commands in a Redis transaction (MULTI/EXEC)
func (c *Client) ExecuteTransaction(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	if c.client == nil {
		return nil, fmt.Errorf("redis client not connected")
	}

	tx := c.client.TxPipeline()
	if err := fn(tx); err != nil {
		return nil, err
	}

	return tx.Exec(ctx)
}
