package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository"
)

type ConversationRepo struct {
	client     *firestore.Client
	collection *firestore.CollectionRef
}

func NewConversationRepository(client *firestore.Client) repositories.ConversationRepository {
	return &ConversationRepo{
		client:     client,
		collection: client.Collection(constants.ConversationsCollection),
	}
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	now := indianstandardtime.Now()
	conversation.CreatedAt = now
	conversation.UpdatedAt = now

	// Generate ObjectID for compatibility
	conversation.ID = primitive.NewObjectID()

	// Use the ObjectID as document ID
	_, err := r.collection.Doc(conversation.ID.Hex()).Set(ctx, conversation)
	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

func (r *ConversationRepo) GetConversationByID(ctx context.Context, id primitive.ObjectID) (*models.Conversation, error) {
	doc, err := r.collection.Doc(id.Hex()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	var conversation models.Conversation
	if err := doc.DataTo(&conversation); err != nil {
		return nil, fmt.Errorf("failed to parse conversation: %w", err)
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

	query := r.collection.
		Where("participants", "array-contains", userID).
		OrderBy("updated_at", firestore.Desc).
		Limit(limit).
		Offset(offset)

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get user conversations: %w", err)
	}

	var conversations []*models.Conversation
	for _, doc := range docs {
		var conversation models.Conversation
		if err := doc.DataTo(&conversation); err != nil {
			continue // Skip malformed documents
		}
		conversations = append(conversations, &conversation)
	}

	return conversations, nil
}

func (r *ConversationRepo) UpdateLastMessageTime(ctx context.Context, conversationID primitive.ObjectID) error {
	now := indianstandardtime.Now()

	_, err := r.collection.Doc(conversationID.Hex()).Update(ctx, []firestore.Update{
		{Path: "last_message_at", Value: now},
		{Path: "updated_at", Value: now},
	})

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return nil
}

func (r *ConversationRepo) DeleteConversation(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.Doc(id.Hex()).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	return nil
}

func (r *ConversationRepo) FindConversationByParticipants(ctx context.Context, participants []string) (*models.Conversation, error) {
	// Firestore approach: query by one participant and filter in memory
	// This is simpler than complex Firestore queries
	conversations, err := r.GetUserConversations(ctx, participants[0], 100, 0)
	if err != nil {
		return nil, err
	}

	for _, conv := range conversations {
		if len(conv.Participants) == len(participants) {
			match := true
			for _, p := range participants {
				found := false
				for _, cp := range conv.Participants {
					if cp == p {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
			}
			if match {
				return conv, nil
			}
		}
	}

	return nil, fmt.Errorf("conversation not found")
}
