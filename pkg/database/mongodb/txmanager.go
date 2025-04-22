package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// TxManager implements database.TxManager for MongoDB
type TxManager struct {
	client *mongo.Client
	logger logging.Logger
}

// NewTxManager creates a new transaction manager
func NewTxManager(client *mongo.Client) *TxManager {
	return &TxManager{
		client: client,
		logger: logging.Get().Named("mongodb-tx"),
	}
}

// WithTransaction executes fn within a MongoDB transaction
func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
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
func (m *TxManager) Begin(ctx context.Context) (context.Context, error) {
	session, err := m.client.StartSession()
	if err != nil {
		return ctx, fmt.Errorf("failed to start MongoDB session: %w", err)
	}

	// Start transaction
	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return ctx, fmt.Errorf("failed to start MongoDB transaction: %w", err)
	}

	// Create session context
	sessCtx := mongo.NewSessionContext(ctx, session)

	// Store session in context
	return context.WithValue(sessCtx, mongoSessionKey, session), nil
}

// Commit commits the MongoDB transaction in the context
func (m *TxManager) Commit(ctx context.Context) error {
	session, ok := ctx.Value(mongoSessionKey).(mongo.Session)
	if !ok || session == nil {
		return fmt.Errorf("MongoDB session not found in context - ensure Begin() was called on this context")
	}

	if err := session.CommitTransaction(ctx); err != nil {
		return fmt.Errorf("failed to commit MongoDB transaction: %w", err)
	}

	session.EndSession(ctx)
	return nil
}

// Rollback aborts the MongoDB transaction in the context
func (m *TxManager) Rollback(ctx context.Context) error {
	session, ok := ctx.Value(mongoSessionKey).(mongo.Session)
	if !ok || session == nil {
		return fmt.Errorf("MongoDB session not found in context - ensure Begin() was called on this context")
	}

	if err := session.AbortTransaction(ctx); err != nil {
		return fmt.Errorf("failed to abort MongoDB transaction: %w", err)
	}

	session.EndSession(ctx)
	return nil
}

// mongoSessionKey is the context key for MongoDB sessions
type mongoSessionKeyType struct{}

var mongoSessionKey = mongoSessionKeyType{}
