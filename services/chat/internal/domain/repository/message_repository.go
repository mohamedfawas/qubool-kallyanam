package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageRepository interface {
	// TODO: Implement message management
	CreateMessage(ctx context.Context, message *models.Message) error
	GetMessagesByConversation(ctx context.Context, conversationID primitive.ObjectID, limit, offset int) ([]*models.Message, error)
	GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error)
	UpdateMessage(ctx context.Context, id primitive.ObjectID, text string) error
	SoftDeleteMessage(ctx context.Context, id primitive.ObjectID) error
	CountMessagesByConversation(ctx context.Context, conversationID primitive.ObjectID) (int, error)
	GetLatestMessageByConversation(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error)
}
