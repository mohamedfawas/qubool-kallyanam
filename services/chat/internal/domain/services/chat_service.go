package services

import (
	"context"
	"errors"
	"sort"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/models"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrUnauthorized         = errors.New("unauthorized access")
	ErrInvalidInput         = errors.New("invalid input")
	ErrDuplicateParticipant = errors.New("cannot create conversation with yourself")
)

type ChatService struct {
	conversationRepo repositories.ConversationRepository
	messageRepo      repositories.MessageRepository
	logger           logging.Logger
}

func NewChatService(
	conversationRepo repositories.ConversationRepository,
	messageRepo repositories.MessageRepository,
	logger logging.Logger,
) *ChatService {
	return &ChatService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		logger:           logger,
	}
}

func (s *ChatService) CreateConversation(ctx context.Context, userID, participantID string) (*models.Conversation, error) {
	s.logger.Info("Creating conversation", "userID", userID, "participantID", participantID)

	// Validate input
	if userID == "" || participantID == "" {
		return nil, ErrInvalidInput
	}

	// Prevent user from creating conversation with themselves
	if userID == participantID {
		return nil, ErrDuplicateParticipant
	}

	// Sort participants to ensure consistent ordering for lookups
	participants := []string{userID, participantID}
	sort.Strings(participants)

	// Check if conversation already exists
	existingConversation, err := s.conversationRepo.FindConversationByParticipants(ctx, participants)
	if err != nil && err != mongo.ErrNoDocuments {
		s.logger.Error("Error checking existing conversation", "error", err)
		return nil, err
	}

	// If conversation exists, return it
	if existingConversation != nil {
		s.logger.Info("Returning existing conversation", "conversationID", existingConversation.ID.Hex())
		return existingConversation, nil
	}

	// Create new conversation
	now := indianstandardtime.Now()
	newConversation := &models.Conversation{
		Participants: participants,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.conversationRepo.CreateConversation(ctx, newConversation); err != nil {
		s.logger.Error("Failed to create conversation", "error", err)
		return nil, err
	}

	s.logger.Info("Created new conversation", "conversationID", newConversation.ID.Hex())
	return newConversation, nil
}

func (s *ChatService) GetUserConversations(ctx context.Context, userID string, limit, offset int) ([]*models.Conversation, int, error) {
	s.logger.Info("Getting user conversations", "userID", userID, "limit", limit, "offset", offset)

	// Validate input
	if userID == "" {
		return nil, 0, ErrInvalidInput
	}

	// Validate and set default pagination
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if limit > 50 {
		limit = 50 // Max limit
	}
	if offset < 0 {
		offset = 0 // Default offset
	}

	// Get conversations for the user
	conversations, err := s.conversationRepo.GetUserConversations(ctx, userID, limit+1, offset) // Get one extra to check if there are more
	if err != nil {
		s.logger.Error("Failed to get user conversations", "error", err)
		return nil, 0, err
	}

	// Calculate total count for pagination
	allConversations, err := s.conversationRepo.GetUserConversations(ctx, userID, 0, 0) // Get all to count
	if err != nil {
		s.logger.Error("Failed to get total conversation count", "error", err)
		allConversations = conversations // Fallback to current batch
	}

	total := len(allConversations)

	// Check if there are more conversations
	hasMore := len(conversations) > limit
	if hasMore {
		conversations = conversations[:limit] // Remove the extra conversation we fetched
	}

	s.logger.Info("Retrieved conversations successfully", "count", len(conversations), "total", total)
	return conversations, total, nil
}

func (s *ChatService) SendMessage(ctx context.Context, userID string, conversationID primitive.ObjectID, text string) (*models.Message, error) {
	s.logger.Info("Sending message", "userID", userID, "conversationID", conversationID.Hex())

	// Validate input
	if userID == "" {
		return nil, ErrInvalidInput
	}

	if text == "" {
		return nil, ErrInvalidInput
	}

	if len(text) > 2000 {
		return nil, ErrInvalidInput
	}

	// Check if conversation exists
	conversation, err := s.conversationRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			s.logger.Error("Conversation not found", "conversationID", conversationID.Hex())
			return nil, ErrConversationNotFound
		}
		s.logger.Error("Error fetching conversation", "error", err)
		return nil, err
	}

	// Check authorization - user must be participant
	isParticipant := false
	for _, participant := range conversation.Participants {
		if participant == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		s.logger.Error("User not authorized to send message", "userID", userID, "conversationID", conversationID.Hex())
		return nil, ErrUnauthorized
	}

	// Create message
	now := indianstandardtime.Now()
	message := &models.Message{
		ConversationID: conversationID,
		SenderID:       userID,
		Text:           text,
		SentAt:         now,
		IsDeleted:      false,
	}

	if err := s.messageRepo.CreateMessage(ctx, message); err != nil {
		s.logger.Error("Failed to create message", "error", err)
		return nil, err
	}

	// Update conversation's last_message_at
	if err := s.conversationRepo.UpdateLastMessageTime(ctx, conversationID); err != nil {
		s.logger.Error("Failed to update last message time", "error", err)
		// Don't fail the request for this non-critical operation
	}

	s.logger.Info("Message sent successfully", "messageID", message.ID.Hex())
	return message, nil
}

func (s *ChatService) GetMessages(ctx context.Context, userID string, conversationID primitive.ObjectID, limit, offset int) ([]*models.Message, int, error) {
	s.logger.Info("Getting messages", "userID", userID, "conversationID", conversationID.Hex(), "limit", limit, "offset", offset)

	// Validate input
	if userID == "" {
		return nil, 0, ErrInvalidInput
	}

	// Validate and set default pagination
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if limit > 50 {
		limit = 50 // Max limit
	}
	if offset < 0 {
		offset = 0 // Default offset
	}

	// Check if conversation exists
	conversation, err := s.conversationRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			s.logger.Error("Conversation not found", "conversationID", conversationID.Hex())
			return nil, 0, ErrConversationNotFound
		}
		s.logger.Error("Error fetching conversation", "error", err)
		return nil, 0, err
	}

	// Check authorization - user must be participant
	isParticipant := false
	for _, participant := range conversation.Participants {
		if participant == userID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		s.logger.Error("User not authorized to view messages", "userID", userID, "conversationID", conversationID.Hex())
		return nil, 0, ErrUnauthorized
	}

	// Get total count for pagination
	total, err := s.messageRepo.CountMessagesByConversation(ctx, conversationID)
	if err != nil {
		s.logger.Error("Failed to get total message count", "error", err)
		total = 0 // Fallback
	}

	// Get messages with pagination
	messages, err := s.messageRepo.GetMessagesByConversation(ctx, conversationID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get messages", "error", err)
		return nil, 0, err
	}

	s.logger.Info("Retrieved messages successfully", "count", len(messages), "total", total)
	return messages, total, nil
}

func (s *ChatService) GetLatestMessage(ctx context.Context, conversationID primitive.ObjectID) (*models.Message, error) {
	return s.messageRepo.GetLatestMessageByConversation(ctx, conversationID)
}

func (s *ChatService) DeleteMessage(ctx context.Context, userID string, messageID primitive.ObjectID) error {
	s.logger.Info("TODO: DeleteMessage", "userID", userID, "messageID", messageID.Hex())
	return errors.New("not implemented yet")
}
