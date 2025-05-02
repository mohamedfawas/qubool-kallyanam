package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisChecker checks Redis database health
type RedisChecker struct {
	client *redis.Client
	name   string
}

// NewRedisChecker creates a new Redis health checker
func NewRedisChecker(client *redis.Client, name string) *RedisChecker {
	return &RedisChecker{
		client: client,
		name:   name,
	}
}

// Check implements the Checker interface
func (c *RedisChecker) Check(ctx context.Context) (Status, error) {
	// Simple ping check for Redis
	_, err := c.client.Ping(ctx).Result()
	if err != nil {
		return StatusNotServing, err
	}

	return StatusServing, nil
}

// Name returns the checker name
func (c *RedisChecker) Name() string {
	return c.name
}
