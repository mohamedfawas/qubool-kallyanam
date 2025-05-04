package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Config struct {
	URI      string
	Database string
	Timeout  time.Duration
}

type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewClient creates a new MongoDB client instance
func NewClient(config *Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.URI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Println("Connected to MongoDB database")
	return &Client{
		client:   client,
		database: client.Database(config.Database),
	}, nil
}

// Close disconnects from the MongoDB server
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// GetCollection returns a handle to the specified collection
func (c *Client) GetCollection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// GetDatabase returns the mongo database
func (c *Client) GetDatabase() *mongo.Database {
	return c.database
}

// GetClient returns the underlying mongo client
func (c *Client) GetClient() *mongo.Client {
	return c.client
}
