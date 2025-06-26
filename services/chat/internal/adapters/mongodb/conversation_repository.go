package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository"
)

type ConversationRepo struct {
	collection *mongo.Collection
}

func NewConversationRepository(db *mongo.Database) repositories.ConversationRepository {
	return &ConversationRepo{
		collection: db.Collection(constants.ConversationsCollection),
	}
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	now := indianstandardtime.Now()
	conversation.CreatedAt = now
	conversation.UpdatedAt = now

	result, err := r.collection.InsertOne(ctx, conversation)
	if err != nil {
		return err
	}

	conversation.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ConversationRepo) GetConversationByID(ctx context.Context, id primitive.ObjectID) (*models.Conversation, error) {
	var conversation models.Conversation

	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepo) GetUserConversations(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error) {
	// Apply default and max limits
	if limit <= 0 {
		limit = constants.DefaultConversationLimit
	}
	if limit > constants.MaxConversationLimit {
		limit = constants.MaxConversationLimit
	}

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
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *ConversationRepo) FindConversationByParticipants(ctx context.Context, participants []string) (*models.Conversation, error) {
	filter := bson.M{
		"participants": bson.M{
			"$all":  participants,
			"$size": len(participants),
		},
	}

	var conversation models.Conversation
	err := r.collection.FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}
