package firestore

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// Config holds Firestore connection configuration values.
type Config struct {
	ProjectID       string
	CredentialsFile string // Path to service account JSON file
	EmulatorHost    string // For development with Firestore emulator
}

// Client wraps the Firestore client instance for later use in our application.
type Client struct {
	client    *firestore.Client
	projectID string
}

// NewClient creates a new Firestore client using the given configuration.
func NewClient(config *Config) (*Client, error) {
	ctx := context.Background()

	var client *firestore.Client
	var err error

	// Check if using emulator (for development)
	if config.EmulatorHost != "" {
		// For emulator, no credentials needed
		client, err = firestore.NewClient(ctx, config.ProjectID)
	} else if config.CredentialsFile != "" {
		// Use service account credentials file
		client, err = firestore.NewClient(ctx, config.ProjectID,
			option.WithCredentialsFile(config.CredentialsFile))
	} else {
		// Use Application Default Credentials (ADC)
		client, err = firestore.NewClient(ctx, config.ProjectID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to Firestore: %w", err)
	}

	log.Println("Connected to Firestore database")
	return &Client{
		client:    client,
		projectID: config.ProjectID,
	}, nil
}

// Close gracefully closes the Firestore connection when the application shuts down.
func (c *Client) Close() error {
	return c.client.Close()
}

// GetClient returns the underlying Firestore client for direct access.
func (c *Client) GetClient() *firestore.Client {
	return c.client
}

// GetCollection returns a handle to the specified collection.
func (c *Client) GetCollection(name string) *firestore.CollectionRef {
	return c.client.Collection(name)
}

// GetProjectID returns the project ID.
func (c *Client) GetProjectID() string {
	return c.projectID
}
