package postgres

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds PostgreSQL connection configuration values.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Client wraps the GORM DB instance for later use in our application.
type Client struct {
	DB *gorm.DB
}

// NewClient creates a new PostgreSQL client using the given configuration.
// It sets up the connection using GORM and applies basic connection pool settings.
func NewClient(config *Config) (*Client, error) {

	// Format the connection string (DSN - Data Source Name) using configuration values
	// Example:
	// host=localhost port=5432 user=postgres password=mypassword dbname=mydb sslmode=disable
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	// GORM configuration: enabling the default logger to show all SQL queries (logger.Info)
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Open the database connection using the DSN and GORM configuration
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Get the generic database object from GORM for low-level settings like connection pooling.
	// Connection pooling is a technique used to manage and reuse database connections efficiently instead of opening and closing a new connection for every query.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB instance: %w", err)
	}

	// SetMaxIdleConns sets the maximum number of idle connections in the pool.
	// Example: Up to 10 connections can remain idle without being closed.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	// Example: If 100 goroutines try to connect, the 101st will wait until one frees up.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// Example: After 1 hour, the connection will be closed and recreated.
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Connected to PostgreSQL database")

	// Return the client instance wrapping the GORM DB
	return &Client{DB: db}, nil
}

// Close gracefully closes the database connection when the application shuts down.
func (c *Client) Close() error {
	// Get the generic DB object from GORM
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %w", err)
	}

	// Close the connection to release resources
	return sqlDB.Close()
}
