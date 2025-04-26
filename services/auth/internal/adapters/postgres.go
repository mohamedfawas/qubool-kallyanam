// services/auth/internal/adapters/postgres.go
package adapters

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
)

// PostgresAdapter provides access to PostgreSQL database
type PostgresAdapter struct {
	client *postgres.Client
	logger logging.Logger
}

// NewPostgresAdapter creates a new PostgreSQL adapter
func NewPostgresAdapter(cfg *config.Config, logger logging.Logger) *PostgresAdapter {
	return &PostgresAdapter{
		client: postgres.NewClient(cfg.Postgres, logger.Named("postgres")),
		logger: logger,
	}
}

// Connect establishes a connection to PostgreSQL
func (a *PostgresAdapter) Connect(ctx context.Context) error {
	return a.client.Connect(ctx)
}

// Close closes the PostgreSQL connection
func (a *PostgresAdapter) Close() error {
	return a.client.Close()
}

// Ping checks the PostgreSQL connection
func (a *PostgresAdapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx)
}

// GetClient returns the underlying PostgreSQL client
func (a *PostgresAdapter) GetClient() *postgres.Client {
	return a.client
}
