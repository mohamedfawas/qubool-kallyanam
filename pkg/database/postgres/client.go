// postgres/client.go
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Config extends the common database config with PostgreSQL specific options
type Config struct {
	database.Config
	SchemaSearchPath string        `yaml:"schema_search_path" json:"schema_search_path"`
	ApplicationName  string        `yaml:"application_name" json:"application_name"`
	StatementTimeout time.Duration `yaml:"statement_timeout" json:"statement_timeout"`

	// Additional connection pool settings
	HealthCheckPeriod time.Duration `yaml:"health_check_period" json:"health_check_period"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime" json:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time" json:"max_conn_idle_time"`
	LazyConnect       bool          `yaml:"lazy_connect" json:"lazy_connect"`
}

// Client is a PostgreSQL client using pgx
type Client struct {
	pool   *pgxpool.Pool
	config Config
	logger logging.Logger
}

// NewClient creates a new PostgreSQL client
func NewClient(config Config, logger logging.Logger) *Client {
	if logger == nil {
		logger = logging.Get().Named("postgres")
	}

	return &Client{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to PostgreSQL
func (c *Client) Connect(ctx context.Context) error {
	connString := c.buildConnectionString()
	c.logger.Debug("Connecting to PostgreSQL",
		logging.String("host", c.config.Host),
		logging.Int("port", c.config.Port),
		logging.String("database", c.config.Database),
		logging.String("user", c.config.Username),
	)

	// Parse connection configuration
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("failed to parse postgres connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(c.config.MaxOpenConns)
	if c.config.MaxIdleConns > 0 {
		poolConfig.MinConns = int32(c.config.MaxIdleConns)
	}

	// Set timeouts and connection parameters
	if c.config.ConnTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = c.config.ConnTimeout
	}
	if c.config.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = c.config.HealthCheckPeriod
	} else {
		poolConfig.HealthCheckPeriod = 1 * time.Minute // reasonable default
	}
	if c.config.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = c.config.MaxConnLifetime
	}
	if c.config.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = c.config.MaxConnIdleTime
	}

	// LazyConnect allows the initial connection to be established when actually needed
	poolConfig.LazyConnect = c.config.LazyConnect

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection unless lazy connect is enabled
	if !c.config.LazyConnect {
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return fmt.Errorf("failed to ping postgres: %w", err)
		}
	}

	c.pool = pool
	c.logger.Info("Connected to PostgreSQL database",
		logging.String("host", c.config.Host),
		logging.Int("port", c.config.Port),
		logging.String("database", c.config.Database),
		logging.String("user", c.config.Username),
	)

	return nil
}

// Close closes the connection pool
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
		c.logger.Info("Closed PostgreSQL connection")
	}
}

// Ping verifies the connection to PostgreSQL
func (c *Client) Ping(ctx context.Context) error {
	if c.pool == nil {
		return fmt.Errorf("postgres client not connected")
	}
	return c.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (c *Client) Stats() pgxpool.Stat {
	if c.pool == nil {
		return pgxpool.Stat{}
	}
	return c.pool.Stat()
}

// Pool returns the underlying connection pool for use with sqlc
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// WithTransaction executes the given function inside a transaction
func (c *Client) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return RunInTx(ctx, c.pool, pgx.TxOptions{}, fn)
}

// WithTxOptions executes the given function inside a transaction with custom options
func (c *Client) WithTxOptions(ctx context.Context, opts pgx.TxOptions, fn func(context.Context) error) error {
	return RunInTx(ctx, c.pool, opts, fn)
}

// buildConnectionString creates a PostgreSQL connection string
func (c *Client) buildConnectionString() string {
	sslMode := c.config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.config.Host,
		c.config.Port,
		c.config.Username,
		c.config.Password,
		c.config.Database,
		sslMode,
	)

	if c.config.ApplicationName != "" {
		connStr += fmt.Sprintf(" application_name=%s", c.config.ApplicationName)
	}

	if c.config.SchemaSearchPath != "" {
		connStr += fmt.Sprintf(" search_path=%s", c.config.SchemaSearchPath)
	}

	if c.config.StatementTimeout > 0 {
		connStr += fmt.Sprintf(" statement_timeout=%d", c.config.StatementTimeout.Milliseconds())
	}

	return connStr
}
