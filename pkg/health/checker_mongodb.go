package health

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBChecker checks MongoDB database health
type MongoDBChecker struct {
	client *mongo.Client
	name   string
}

// NewMongoDBChecker creates a new MongoDB health checker
func NewMongoDBChecker(client *mongo.Client, name string) *MongoDBChecker {
	return &MongoDBChecker{
		client: client,
		name:   name,
	}
}

// Check implements the Checker interface
func (c *MongoDBChecker) Check(ctx context.Context) (Status, error) {
	err := c.client.Ping(ctx, readpref.Primary())
	if err != nil {
		return StatusNotServing, err
	}
	return StatusServing, nil
}

// Name returns the checker name
func (c *MongoDBChecker) Name() string {
	return c.name
}
