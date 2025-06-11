package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConversationRepository interface {
	// TODO: Implement conversation management
	CreateConversation(ctx context.Context, conversation *models.Conversation) error
	GetConversationByID(ctx context.Context, id primitive.ObjectID) (*models.Conversation, error)
	GetUserConversations(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, error)
	UpdateLastMessageTime(ctx context.Context, conversationID primitive.ObjectID) error
	DeleteConversation(ctx context.Context, id primitive.ObjectID) error
	FindConversationByParticipants(ctx context.Context, participants []string) (*models.Conversation, error)
}
