package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// TxManager implements database.TxManager for PostgreSQL
type TxManager struct {
	pool   *pgxpool.Pool
	logger logging.Logger
}

// NewTxManager creates a new transaction manager
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{
		pool:   pool,
		logger: logging.Get().Named("postgres-tx"),
	}
}

// Begin starts a new transaction
func (m *TxManager) Begin(ctx context.Context) (context.Context, error) {
	// Check if transaction already exists in context
	if tx := GetTx(ctx); tx != nil {
		return ctx, nil // Transaction already started
	}

	// Start a transaction
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed to begin PostgreSQL transaction: %w", err)
	}

	// Store the transaction in context
	return WithTx(ctx, tx), nil
}

// Commit commits the transaction in the context
func (m *TxManager) Commit(ctx context.Context) error {
	tx := GetTx(ctx)
	if tx == nil {
		return fmt.Errorf("PostgreSQL transaction not found in context - ensure Begin() was called on this context")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit PostgreSQL transaction: %w", err)
	}

	return nil
}

// Rollback rolls back the transaction in the context
func (m *TxManager) Rollback(ctx context.Context) error {
	tx := GetTx(ctx)
	if tx == nil {
		return fmt.Errorf("PostgreSQL transaction not found in context - ensure Begin() was called on this context")
	}

	if err := tx.Rollback(ctx); err != nil {
		// Don't wrap this error, as pgx returns a specific error when a transaction
		// has already been committed/rolled back
		return err
	}

	return nil
}

// WithTransaction executes fn within a transaction
func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Start transaction
	txCtx, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	// Ensure transaction is terminated
	defer func() {
		if tx := GetTx(txCtx); tx != nil {
			// If we reach here and the transaction is still active, roll it back
			_ = tx.Rollback(ctx)
		}
	}()

	// Execute function
	if err := fn(txCtx); err != nil {
		if tx := GetTx(txCtx); tx != nil {
			_ = tx.Rollback(ctx)
		}
		return err
	}

	// Commit transaction
	return m.Commit(txCtx)
}
