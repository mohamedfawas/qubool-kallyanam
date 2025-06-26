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

type MessageRepo struct {
	collection *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) repositories.MessageRepository {
	return &MessageRepo{
		collection: db.Collection(constants.MessagesCollection),
	}
}

func (r *MessageRepo) CreateMessage(ctx context.Context, message *models.Message) error {
	now := indianstandardtime.Now()
	message.SentAt = now
	message.IsDeleted = false

	result, err := r.collection.InsertOne(ctx, message)
	if err != nil {
		return err
	}

	message.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MessageRepo) GetMessagesByConversation(ctx context.Context, conversationID primitive.ObjectID, limit, offset int) ([]*models.Message, error) {
	// Apply default and max limits
	if limit <= 0 {
		limit = constants.DefaultMessageLimit
	}
	if limit > constants.MaxMessageLimit {
		limit = constants.MaxMessageLimit
	}

	filter := bson.M{
		"conversation_id": conversationID,
		"is_deleted":      false,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "sent_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}

	return messages, cursor.Err()
}

func (r *MessageRepo) GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	var message models.Message
	filter := bson.M{
		"_id":        id,
		"is_deleted": false,
	}

	err := r.collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepo) UpdateMessage(ctx context.Context, id primitive.ObjectID, text string) error {
	update := bson.M{
		"$set": bson.M{
			"text": text,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "is_deleted": false}, update)
	return err
}

func (r *MessageRepo) SoftDeleteMessage(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *MessageRepo) CountMessagesByConversation(ctx context.Context, conversationID primitive.ObjectID) (int, error) {
	filter := bson.M{
		"conversation_id": conversationID,
		"is_deleted":      false,
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func (r *MessageRepo) GetLatestMessageByConversation(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error) {
	filter := bson.M{
		"conversation_id": conversationID,
		"is_deleted":      false,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "sent_at", Value: -1}})

	var message models.Message
	err := r.collection.FindOne(ctx, filter, opts).Decode(&message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}
