package postgres

import (
	"fmt"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/log"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string
	MaxConns int
	MaxIdle  int
	Timeout  time.Duration
}

// Client represents a PostgreSQL database client
type Client struct {
	DB     *gorm.DB
	logger *zap.Logger
}

// NewClient creates a new PostgreSQL client
func NewClient(cfg Config, serviceName string, zapLogger *zap.Logger) (*Client, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Configure GORM logger
	gormLogLevel := logger.Silent
	if zapLogger.Core().Enabled(zap.DebugLevel) {
		gormLogLevel = logger.Info
	}

	gormLogger := logger.New(
		log.NewGormLogAdapter(zapLogger),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetConnMaxLifetime(cfg.Timeout)

	zapLogger.Info("Connected to PostgreSQL database",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.String("service", serviceName))

	return &Client{
		DB:     db,
		logger: zapLogger,
	}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive
func (c *Client) Ping() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
