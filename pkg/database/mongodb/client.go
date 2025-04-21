package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"qubool-kallyanam/pkg/database"
	"qubool-kallyanam/pkg/logging"
)

// Config extends the common database config with MongoDB specific options
type Config struct {
	database.Config
	ReplicaSet          string        `yaml:"replica_set"`
	AuthSource          string        `yaml:"auth_source"`
	ConnectTimeout      time.Duration `yaml:"connect_timeout"`
	ServerSelectionTime time.Duration `yaml:"server_selection_timeout"`
	MaxPoolSize         uint64        `yaml:"max_pool_size"`
	MinPoolSize         uint64        `yaml:"min_pool_size"`
}

// Client is a MongoDB database client
type Client struct {
	client *mongo.Client
	config Config
	logger logging.Logger
	db     *mongo.Database
}

// NewClient creates a new MongoDB client
func NewClient(config Config, logger logging.Logger) *Client {
	if logger == nil {
		logger = logging.Get().Named("mongodb")
	}

	return &Client{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to MongoDB
func (c *Client) Connect(ctx context.Context) error {
	uri := c.buildConnectionString()

	// Setup client options
	opts := options.Client()
	opts.ApplyURI(uri)

	// Configure connection pool
	if c.config.MaxPoolSize > 0 {
		opts.SetMaxPoolSize(c.config.MaxPoolSize)
	}
	if c.config.MinPoolSize > 0 {
		opts.SetMinPoolSize(c.config.MinPoolSize)
	}

	// Configure timeouts
	if c.config.ConnectTimeout > 0 {
		opts.SetConnectTimeout(c.config.ConnectTimeout)
	}
	if c.config.ServerSelectionTime > 0 {
		opts.SetServerSelectionTimeout(c.config.ServerSelectionTime)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}

	c.client = client
	c.db = client.Database(c.config.Database)

	c.logger.Info("Connected to MongoDB database",
		logging.String("host", c.config.Host),
		logging.Int("port", c.config.Port),
		logging.String("database", c.config.Database),
	)

	return nil
}

// Close closes the MongoDB connection
func (c *Client) Close(ctx context.Context) error {
	if c.client != nil {
		if err := c.client.Disconnect(ctx); err != nil {
			return fmt.Errorf("failed to disconnect from mongodb: %w", err)
		}
		c.logger.Info("Closed MongoDB connection")
	}
	return nil
}

// Ping verifies the connection to MongoDB
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("mongodb client not connected")
	}
	return c.client.Ping(ctx, readpref.Primary())
}

// Stats returns connection pool statistics
func (c *Client) Stats() interface{} {
	// MongoDB doesn't expose pool stats directly
	return map[string]interface{}{
		"connected": c.client != nil,
	}
}

// GetClient returns the underlying MongoDB client
func (c *Client) GetClient() *mongo.Client {
	return c.client
}

// GetDatabase returns the MongoDB database
func (c *Client) GetDatabase() *mongo.Database {
	return c.db
}

// Collection returns a handle to the specified collection
func (c *Client) Collection(name string) *mongo.Collection {
	return c.db.Collection(name)
}

// buildConnectionString creates a MongoDB connection string
func (c *Client) buildConnectionString() string {
	auth := ""
	if c.config.Username != "" && c.config.Password != "" {
		auth = fmt.Sprintf("%s:%s@", c.config.Username, c.config.Password)
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d/%s",
		auth,
		c.config.Host,
		c.config.Port,
		c.config.Database,
	)

	// Add options
	hasOptions := false
	options := ""

	if c.config.ReplicaSet != "" {
		options += fmt.Sprintf("replicaSet=%s", c.config.ReplicaSet)
		hasOptions = true
	}

	if c.config.AuthSource != "" {
		if hasOptions {
			options += "&"
		}
		options += fmt.Sprintf("authSource=%s", c.config.AuthSource)
		hasOptions = true
	}

	if hasOptions {
		uri += "?" + options
	}

	return uri
}
