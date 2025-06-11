package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository"
)

type ConversationRepo struct {
	collection *mongo.Collection
}

func NewConversationRepository(db *mongo.Database) repositories.ConversationRepository {
	return &ConversationRepo{
		collection: db.Collection(models.Conversation{}.CollectionName()),
	}
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	now := indianstandardtime.Now()
	conversation.CreatedAt = now
	conversation.UpdatedAt = now

	// Insert the conversation document into the MongoDB collection.
	// This returns a result object that contains the inserted ID.
	result, err := r.collection.InsertOne(ctx, conversation)
	if err != nil {
		return err
	}

	// Assign the MongoDB-generated ObjectID back to the conversation struct.
	// Type assertion is needed because InsertedID is of type interface{}.
	// Example: InsertedID = ObjectID("665f8e61d9d3a12c43a5b52e")
	conversation.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ConversationRepo) GetConversationByID(ctx context.Context, id primitive.ObjectID) (*models.Conversation, error) {
	var conversation models.Conversation

	// Perform a MongoDB query to find one document where "_id" matches the given ID.
	// bson.M is a map used to construct MongoDB queries in Go.
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepo) GetUserConversations(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	// Define a filter to match conversations where the user is one of the participants.
	// MongoDB will match any document where `participants` array contains the given userID.
	filter := bson.M{"participants": userID}
	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []*models.Conversation
	for cursor.Next(ctx) {
		var conversation models.Conversation
		// Decode each document into a Conversation struct
		if err := cursor.Decode(&conversation); err != nil {
			return nil, err
		}
		conversations = append(conversations, &conversation)
	}

	return conversations, cursor.Err()
}

func (r *ConversationRepo) UpdateLastMessageTime(ctx context.Context, conversationID primitive.ObjectID) error {
	now := indianstandardtime.Now()
	update := bson.M{
		"$set": bson.M{
			"last_message_at": now,
			"updated_at":      now,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": conversationID}, update)
	return err
}

func (r *ConversationRepo) DeleteConversation(ctx context.Context, id primitive.ObjectID) error {
	// TODO: Implement conversation deletion
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *ConversationRepo) FindConversationByParticipants(ctx context.Context, participants []string) (*models.Conversation, error) {
	// Construct a MongoDB filter to search for conversations.
	// We're looking for documents where:
	// 1. The "participants" field contains all the items in the input slice (`$all`).
	// 2. The number of participants is exactly equal to the length of the input slice (`$size`).
	// This ensures we only match conversations with *exactly* the same participants (no more, no less).
	filter := bson.M{
		"participants": bson.M{
			"$all":  participants,
			"$size": len(participants),
		},
	}

	var conversation models.Conversation

	// Perform a MongoDB query on the collection to find one document matching the filter
	// Decode the result into the `conversation` variable
	err := r.collection.FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}
