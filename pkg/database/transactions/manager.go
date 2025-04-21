package transactions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"

	"qubool-kallyanam/pkg/database/postgres"
	"qubool-kallyanam/pkg/logging"
)

// PostgresTxManager implements transaction management for PostgreSQL
type PostgresTxManager struct {
	pool   *pgxpool.Pool
	logger logging.Logger
}

// NewPostgresTxManager creates a new PostgreSQL transaction manager
func NewPostgresTxManager(pool *pgxpool.Pool, logger logging.Logger) *PostgresTxManager {
	if logger == nil {
		logger = logging.Get().Named("postgres-tx")
	}

	return &PostgresTxManager{
		pool:   pool,
		logger: logger,
	}
}

// Begin starts a new transaction and stores it in the context
func (m *PostgresTxManager) Begin(ctx context.Context) (context.Context, error) {
	// Check if transaction already exists in context
	if postgres.GetTx(ctx) != nil {
		return ctx, nil // Transaction already started
	}

	// Start a transaction
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Store the transaction in context
	return postgres.WithTx(ctx, tx), nil
}

// Commit commits the transaction in the context
func (m *PostgresTxManager) Commit(ctx context.Context) error {
	tx := postgres.GetTx(ctx)
	if tx == nil {
		return fmt.Errorf("no transaction found in context")
	}

	return tx.Commit(ctx)
}

// Rollback rolls back the transaction in the context
func (m *PostgresTxManager) Rollback(ctx context.Context) error {
	tx := postgres.GetTx(ctx)
	if tx == nil {
		return fmt.Errorf("no transaction found in context")
	}

	return tx.Rollback(ctx)
}

// WithTransaction executes fn within a transaction
func (m *PostgresTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Start transaction
	txCtx, err := m.Begin(ctx)
	if err != nil {
		return err
	}

	// Ensure transaction is terminated
	defer func() {
		if tx := postgres.GetTx(txCtx); tx != nil {
			// If we reach here and the transaction is still active, roll it back
			_ = tx.Rollback(ctx)
		}
	}()

	// Execute function
	if err := fn(txCtx); err != nil {
		if tx := postgres.GetTx(txCtx); tx != nil {
			_ = tx.Rollback(ctx)
		}
		return err
	}

	// Commit transaction
	return m.Commit(txCtx)
}

// MongoTxManager implements transaction management for MongoDB
type MongoTxManager struct {
	client *mongo.Client
	logger logging.Logger
}

// NewMongoTxManager creates a new MongoDB transaction manager
func NewMongoTxManager(client *mongo.Client, logger logging.Logger) *MongoTxManager {
	if logger == nil {
		logger = logging.Get().Named("mongo-tx")
	}

	return &MongoTxManager{
		client: client,
		logger: logger,
	}
}

// WithTransaction executes fn within a MongoDB transaction
func (m *MongoTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// MongoDB sessions can't be easily stored in context like PostgreSQL transactions
	// So we'll use the built-in WithSession and StartTransaction APIs

	// Start a session
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	// Run the transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := fn(sessCtx); err != nil {
			return nil, err
		}
		return nil, nil
	})

	return err
}

// Begin starts a new MongoDB session and transaction
// Implementation for interface compatibility, but WithTransaction is preferred
func (m *MongoTxManager) Begin(ctx context.Context) (context.Context, error) {
	session, err := m.client.StartSession()
	if err != nil {
		return ctx, fmt.Errorf("failed to start mongo session: %w", err)
	}

	// Start transaction
	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return ctx, fmt.Errorf("failed to start mongo transaction: %w", err)
	}

	// Create session context
	sessCtx := mongo.NewSessionContext(ctx, session)

	// Store session in context
	return context.WithValue(sessCtx, mongoSessionKey, session), nil
}

// Commit commits the MongoDB transaction in the context
func (m *MongoTxManager) Commit(ctx context.Context) error {
	session, ok := ctx.Value(mongoSessionKey).(mongo.Session)
	if !ok || session == nil {
		return fmt.Errorf("no mongo session found in context")
	}

	if err := session.CommitTransaction(ctx); err != nil {
		return fmt.Errorf("failed to commit mongo transaction: %w", err)
	}

	return nil
}

// Rollback aborts the MongoDB transaction in the context
func (m *MongoTxManager) Rollback(ctx context.Context) error {
	session, ok := ctx.Value(mongoSessionKey).(mongo.Session)
	if !ok || session == nil {
		return fmt.Errorf("no mongo session found in context")
	}

	if err := session.AbortTransaction(ctx); err != nil {
		return fmt.Errorf("failed to abort mongo transaction: %w", err)
	}

	return nil
}

// mongoSessionKey is the context key for MongoDB sessions
type mongoSessionKeyType struct{}

var mongoSessionKey = mongoSessionKeyType{}
