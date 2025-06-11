package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation represents a chat conversation between users
type Conversation struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Participants  []string           `bson:"participants" json:"participants"` // Array of user IDs
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	LastMessageAt *time.Time         `bson:"last_message_at,omitempty" json:"last_message_at,omitempty"`
}

// CollectionName returns the collection name for MongoDB
func (Conversation) CollectionName() string {
	return "conversations"
}
