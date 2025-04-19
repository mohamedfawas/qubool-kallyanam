// pkg/db/manager.go
package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/log"
	"go.uber.org/zap"
)

// Manager handles database connections and provides a unified interface
type Manager struct {
	logger       *log.Logger
	pgClients    map[string]*postgres.Client
	redisClients map[string]*redis.Client
	mongoClients map[string]*mongodb.Client
	pgMutex      sync.RWMutex
	redisMutex   sync.RWMutex
	mongoMutex   sync.RWMutex
	serviceName  string
}

// NewManager creates a new database manager
func NewManager(logger *log.Logger, serviceName string) *Manager {
	return &Manager{
		logger:       logger.WithComponent("db-manager"),
		pgClients:    make(map[string]*postgres.Client),
		redisClients: make(map[string]*redis.Client),
		mongoClients: make(map[string]*mongodb.Client),
		serviceName:  serviceName,
	}
}

// GetPostgresClient returns a PostgreSQL client, creating it if it doesn't exist
func (m *Manager) GetPostgresClient(ctx context.Context, cfg postgres.Config, name string) (*postgres.Client, error) {
	// First, try to get an existing client
	m.pgMutex.RLock()
	client, exists := m.pgClients[name]
	m.pgMutex.RUnlock()

	if exists {
		// Make sure the connection is still valid
		if err := client.Ping(); err == nil {
			return client, nil
		}
		m.logger.Warn("PostgreSQL connection is stale, reconnecting",
			zap.String("name", name),
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port))
	}

	// Get write lock to create a new client
	m.pgMutex.Lock()
	defer m.pgMutex.Unlock()

	// Check again in case another goroutine created the client while we were waiting
	if client, exists = m.pgClients[name]; exists {
		if err := client.Ping(); err == nil {
			return client, nil
		}
	}

	// Create a new client
	zapLogger := m.logger.Logger
	client, err := postgres.NewClient(cfg, m.serviceName, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL client %s: %w", name, err)
	}

	// Store the client for reuse
	m.pgClients[name] = client
	return client, nil
}

// GetRedisClient returns a Redis client, creating it if it doesn't exist
func (m *Manager) GetRedisClient(ctx context.Context, cfg redis.Config, name string) (*redis.Client, error) {
	// First, try to get an existing client
	m.redisMutex.RLock()
	client, exists := m.redisClients[name]
	m.redisMutex.RUnlock()

	if exists {
		// Make sure the connection is still valid
		if err := client.Ping(ctx); err == nil {
			return client, nil
		}
		m.logger.Warn("Redis connection is stale, reconnecting",
			zap.String("name", name),
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port))
	}

	// Get write lock to create a new client
	m.redisMutex.Lock()
	defer m.redisMutex.Unlock()

	// Check again in case another goroutine created the client while we were waiting
	if client, exists = m.redisClients[name]; exists {
		if err := client.Ping(ctx); err == nil {
			return client, nil
		}
	}

	// Create a new client
	zapLogger := m.logger.Logger
	client, err := redis.NewClient(ctx, cfg, m.serviceName, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client %s: %w", name, err)
	}

	// Store the client for reuse
	m.redisClients[name] = client
	return client, nil
}

// GetMongoDBClient returns a MongoDB client, creating it if it doesn't exist
func (m *Manager) GetMongoDBClient(ctx context.Context, cfg mongodb.Config, name string) (*mongodb.Client, error) {
	// First, try to get an existing client
	m.mongoMutex.RLock()
	client, exists := m.mongoClients[name]
	m.mongoMutex.RUnlock()

	if exists {
		// Make sure the connection is still valid
		if err := client.Ping(ctx); err == nil {
			return client, nil
		}
		m.logger.Warn("MongoDB connection is stale, reconnecting",
			zap.String("name", name),
			zap.String("uri", cfg.URI),
			zap.String("database", cfg.Database))
	}

	// Get write lock to create a new client
	m.mongoMutex.Lock()
	defer m.mongoMutex.Unlock()

	// Check again in case another goroutine created the client while we were waiting
	if client, exists = m.mongoClients[name]; exists {
		if err := client.Ping(ctx); err == nil {
			return client, nil
		}
	}

	// Create a new client
	zapLogger := m.logger.Logger
	client, err := mongodb.NewClient(ctx, cfg, m.serviceName, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client %s: %w", name, err)
	}

	// Store the client for reuse
	m.mongoClients[name] = client
	return client, nil
}

// Close gracefully shuts down all database connections
func (m *Manager) Close(ctx context.Context) error {
	var errors []error

	// Close PostgreSQL connections
	m.pgMutex.RLock()
	for name, client := range m.pgClients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close PostgreSQL client %s: %w", name, err))
		}
	}
	m.pgMutex.RUnlock()

	// Close Redis connections
	m.redisMutex.RLock()
	for name, client := range m.redisClients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis client %s: %w", name, err))
		}
	}
	m.redisMutex.RUnlock()

	// Close MongoDB connections
	m.mongoMutex.RLock()
	for name, client := range m.mongoClients {
		if err := client.Close(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to close MongoDB client %s: %w", name, err))
		}
	}
	m.mongoMutex.RUnlock()

	if len(errors) > 0 {
		// Log all errors but return only the first one
		for _, err := range errors {
			m.logger.Error("Error closing database connection", zap.Error(err))
		}
		return errors[0]
	}

	return nil
}
