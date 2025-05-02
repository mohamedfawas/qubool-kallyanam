package health

import (
	"context"

	"gorm.io/gorm"
)

// PostgresChecker checks PostgreSQL database health
type PostgresChecker struct {
	db   *gorm.DB
	name string
}

// NewPostgresChecker creates a new PostgreSQL health checker
func NewPostgresChecker(db *gorm.DB, name string) *PostgresChecker {
	return &PostgresChecker{
		db:   db,
		name: name,
	}
}

// Check implements the Checker interface
func (c *PostgresChecker) Check(ctx context.Context) (Status, error) {
	sqlDB, err := c.db.DB()
	if err != nil {
		return StatusNotServing, err
	}

	// Use context for timeout
	err = sqlDB.PingContext(ctx)
	if err != nil {
		return StatusNotServing, err
	}

	return StatusServing, nil
}

// Name returns the checker name
func (c *PostgresChecker) Name() string {
	return c.name
}
