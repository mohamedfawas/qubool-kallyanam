package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Config extends the common database config with PostgreSQL specific options
type Config struct {
	database.Config
	SchemaSearchPath string        `yaml:"schema_search_path"`
	ApplicationName  string        `yaml:"application_name"`
	StatementTimeout time.Duration `yaml:"statement_timeout"`
}

// Client is a PostgreSQL client using pgx
type Client struct {
	pool   *pgxpool.Pool
	config Config
	logger logging.Logger
}

// NewClient creates a new PostgreSQL client
func NewClient(config Config, logger logging.Logger) *Client {
	// Set default values if not specified
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 10 // Default max open connections
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 5 // Default max idle connections
	}
	if config.ConnTimeout <= 0 {
		config.ConnTimeout = 5 * time.Second // Default connection timeout
	}
	if config.QueryTimeout <= 0 {
		config.QueryTimeout = 30 * time.Second // Default query timeout
	}
	if config.SSLMode == "" {
		config.SSLMode = "disable" // Default SSL mode
	}
	if config.Port <= 0 {
		config.Port = 5432 // Default PostgreSQL port
	}

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

	// Set reasonable defaults for timeouts
	if c.config.ConnTimeout > 0 {
		poolConfig.ConnConfig.ConnectTimeout = c.config.ConnTimeout
	}

	// Create the connection pool
	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
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
func (c *Client) Close() error {
	if c.pool != nil {
		c.pool.Close()
		c.logger.Info("Closed PostgreSQL connection")
	}
	return nil
}

// Ping verifies the connection to PostgreSQL
func (c *Client) Ping(ctx context.Context) error {
	if c.pool == nil {
		return fmt.Errorf("postgres client not connected")
	}
	return c.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (c *Client) Stats() interface{} {
	if c.pool == nil {
		return nil
	}
	return c.pool.Stat()
}

// GetPool returns the underlying connection pool for use with sqlc
func (c *Client) GetPool() *pgxpool.Pool {
	return c.pool
}

// buildConnectionString creates a PostgreSQL connection string
func (c *Client) buildConnectionString() string {
	// Check for environment variables first (industry standard practice)
	host := c.config.Host
	if envHost := os.Getenv("POSTGRES_HOST"); envHost != "" {
		host = envHost
	}

	port := c.config.Port
	if envPort := os.Getenv("POSTGRES_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
		}
	}

	username := c.config.Username
	if envUser := os.Getenv("POSTGRES_USER"); envUser != "" {
		username = envUser
	}

	password := c.config.Password
	if envPass := os.Getenv("POSTGRES_PASSWORD"); envPass != "" {
		password = envPass
	}

	database := c.config.Database
	if envDB := os.Getenv("POSTGRES_DB"); envDB != "" {
		database = envDB
	}

	sslMode := c.config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	// Build connection string with proper parameter formatting
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		username,
		password,
		database,
		sslMode,
	)

	// Add other optional parameters if needed
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
