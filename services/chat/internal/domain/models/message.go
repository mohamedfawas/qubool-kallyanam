package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message represents a chat message
type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID primitive.ObjectID `bson:"conversation_id" json:"conversation_id"`
	SenderID       string             `bson:"sender_id" json:"sender_id"`
	Text           string             `bson:"text" json:"text"`
	SentAt         time.Time          `bson:"sent_at" json:"sent_at"`
	IsDeleted      bool               `bson:"is_deleted" json:"is_deleted"`
}

// CollectionName returns the collection name for MongoDB
func (Message) CollectionName() string {
	return "messages"
}
