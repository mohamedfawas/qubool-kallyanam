package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string // Redis server hostname (e.g., "localhost")
	Port     string // Redis server port (e.g., "6379")
	Password string // Redis password, if authentication is enabled
	DB       int    // Redis database index (default is 0)
}

// Client wraps the go-redis client to expose our own simplified interface
type Client struct {
	client *redis.Client
}

func NewClient(config *Config) (*Client, error) {
	// Step 1: Create a new Redis client using the configuration
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Step 2: Create a context with timeout to avoid hanging if Redis is unreachable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 3: Ping Redis to confirm the connection is successful
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Step 4: Return the custom Client wrapper
	return &Client{client: client}, nil
}

// Close gracefully closes the Redis client connection.
// Example use: defer redisClient.Close()
func (c *Client) Close() error {
	return c.client.Close()
}

// Set stores a key-value pair in Redis with an optional expiration duration.
// If expiration is 0, the key does not expire.
// Example: Set(ctx, "username", "alice", 10*time.Second)
// This will store "username" = "alice" that expires in 10 seconds.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Del removes a key from Redis
func (c *Client) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// TTL returns the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// GetClient returns the underlying redis client for use with health checkers
func (c *Client) GetClient() *redis.Client {
	return c.client
}
