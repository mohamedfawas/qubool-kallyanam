package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

type Config struct {
	URI      string
	Database string
	Username string
	Password string
	MaxConns uint64
	MinConns uint64
	Timeout  time.Duration
}

// Client represents a MongoDB client
type Client struct {
	client     *mongo.Client
	database   *mongo.Database
	logger     *zap.Logger
	dbName     string
	collection map[string]*mongo.Collection
}

// NewClient creates a new MongoDB client
func NewClient(ctx context.Context, cfg Config, serviceName string, logger *zap.Logger) (*Client, error) {
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxConns).
		SetMinPoolSize(cfg.MinConns).
		SetMaxConnecting(uint64(10))

	if cfg.Username != "" && cfg.Password != "" {
		clientOptions.SetAuth(options.Credential{
			Username: cfg.Username,
			Password: cfg.Password,
		})
	}

	connectCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	database := client.Database(cfg.Database)

	logger.Info("Connected to MongoDB database",
		zap.String("uri", cfg.URI),
		zap.String("database", cfg.Database),
		zap.String("service", serviceName))

	return &Client{
		client:     client,
		database:   database,
		logger:     logger,
		dbName:     cfg.Database,
		collection: make(map[string]*mongo.Collection),
	}, nil
}

// Close disconnects from mongodb
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// Ping checks if the mongodb connection is alive
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

// Collection returns a mongodb collection
func (c *Client) Collection(name string) *mongo.Collection {
	if col, ok := c.collection[name]; ok {
		return col
	}

	c.collection[name] = c.database.Collection(name)
	return c.collection[name]
}

// Database returns the mongodb database
func (c *Client) Database() *mongo.Database {
	return c.database
}
