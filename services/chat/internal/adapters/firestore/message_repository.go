package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository"
)

type MessageRepo struct {
	client     *firestore.Client
	collection *firestore.CollectionRef
}

func NewMessageRepository(client *firestore.Client) repositories.MessageRepository {
	return &MessageRepo{
		client:     client,
		collection: client.Collection(constants.MessagesCollection),
	}
}

func (r *MessageRepo) CreateMessage(ctx context.Context, message *models.Message) error {
	now := indianstandardtime.Now()
	message.SentAt = now
	message.IsDeleted = false

	// Generate ObjectID for compatibility
	message.ID = primitive.NewObjectID()

	// Use the ObjectID as document ID
	_, err := r.collection.Doc(message.ID.Hex()).Set(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

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

	query := r.collection.
		Where("conversation_id", "==", conversationID).
		Where("is_deleted", "==", false).
		OrderBy("sent_at", firestore.Desc).
		Limit(limit).
		Offset(offset)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var messages []*models.Message
	for _, doc := range docs {
		var message models.Message
		if err := doc.DataTo(&message); err != nil {
			continue // Skip malformed documents
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

func (r *MessageRepo) GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	doc, err := r.collection.Doc(id.Hex()).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var message models.Message
	if err := doc.DataTo(&message); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	if message.IsDeleted {
		return nil, fmt.Errorf("message not found")
	}

	return &message, nil
}

func (r *MessageRepo) UpdateMessage(ctx context.Context, id primitive.ObjectID, text string) error {
	_, err := r.collection.Doc(id.Hex()).Update(ctx, []firestore.Update{
		{Path: "text", Value: text},
	})
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}
	return nil
}

func (r *MessageRepo) SoftDeleteMessage(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.Doc(id.Hex()).Update(ctx, []firestore.Update{
		{Path: "is_deleted", Value: true},
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

func (r *MessageRepo) CountMessagesByConversation(ctx context.Context, conversationID primitive.ObjectID) (int, error) {
	query := r.collection.
		Where("conversation_id", "==", conversationID).
		Where("is_deleted", "==", false)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return len(docs), nil
}

func (r *MessageRepo) GetLatestMessageByConversation(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error) {
	query := r.collection.
		Where("conversation_id", "==", conversationID).
		Where("is_deleted", "==", false).
		OrderBy("sent_at", firestore.Desc).
		Limit(1)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest message: %w", err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no messages found")
	}

	var message models.Message
	if err := docs[0].DataTo(&message); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &message, nil
}
