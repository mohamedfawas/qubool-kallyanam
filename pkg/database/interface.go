package database

import (
	"context"
	"time"
)

// Config contains common database configuration
type Config struct {
	Host          string        `yaml:"host"`
	Port          int           `yaml:"port"`
	Username      string        `yaml:"username"`
	Password      string        `yaml:"password"`
	Database      string        `yaml:"database"`
	MaxOpenConns  int           `yaml:"max_open_conns"`
	MaxIdleConns  int           `yaml:"max_idle_conns"`
	ConnTimeout   time.Duration `yaml:"conn_timeout"`
	QueryTimeout  time.Duration `yaml:"query_timeout"`
	RetryAttempts int           `yaml:"retry_attempts"`
	SSLMode       string        `yaml:"ssl_mode"`
}

// Client defines a generic database client interface
type Client interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context) error

	// Close closes all connections to the database
	Close() error

	// Ping checks if the database is accessible
	Ping(ctx context.Context) error

	// Stats returns information about the connection pool
	Stats() interface{}
}

// Repository is a generic interface for data access operations
type Repository interface {
	// WithTransaction executes operations within a transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// TxManager defines the transaction manager interface
type TxManager interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) (context.Context, error)

	// Commit commits the transaction associated with the context
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction associated with the context
	Rollback(ctx context.Context) error

	// WithTransaction executes the provided function within a transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
