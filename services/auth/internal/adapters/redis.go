// services/auth/internal/adapters/redis.go
package adapters

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
)

// RedisAdapter provides access to Redis
type RedisAdapter struct {
	client *redis.Client
	logger logging.Logger
}

// NewRedisAdapter creates a new Redis adapter
func NewRedisAdapter(cfg *config.Config, logger logging.Logger) *RedisAdapter {
	return &RedisAdapter{
		client: redis.NewClient(cfg.Redis, logger.Named("redis")),
		logger: logger,
	}
}

// Connect establishes a connection to Redis
func (a *RedisAdapter) Connect(ctx context.Context) error {
	return a.client.Connect(ctx)
}

// Close closes the Redis connection
func (a *RedisAdapter) Close() error {
	return a.client.Close()
}

// Ping checks the Redis connection
func (a *RedisAdapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx)
}

// GetClient returns the underlying Redis client
func (a *RedisAdapter) GetClient() *redis.Client {
	return a.client
}
