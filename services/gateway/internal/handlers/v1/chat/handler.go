package chat

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/chat"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type Handler struct {
	chatClient *chat.Client
	userClient *user.Client
	logger     logging.Logger
	hub        *Hub
}

func NewHandler(chatClient *chat.Client, userClient *user.Client, logger logging.Logger) *Handler {
	hub := NewHub(logger)

	// Start the hub in a separate goroutine
	go func() {
		hub.Run()
	}()

	return &Handler{
		chatClient: chatClient,
		userClient: userClient,
		logger:     logger,
		hub:        hub,
	}
}

type CreateConversationRequest struct {
	RecipientID uint64 `json:"recipient_id" binding:"required"`
}

type SendMessageRequest struct {
	Text string `json:"text" binding:"required"`
}

type CreateConversationResponse struct {
	ConversationID string          `json:"conversation_id"`
	Participant    ParticipantInfo `json:"participant"`
	CreatedAt      string          `json:"created_at"`
	LastMessageAt  *string         `json:"last_message_at"`
}

type ParticipantInfo struct {
	ID                uint64  `json:"id"`
	Name              string  `json:"name"`
	ProfilePictureURL *string `json:"profile_picture_url"`
}

type MessageInfo struct {
	MessageID string `json:"message_id"`
	Text      string `json:"text"`
	SenderID  string `json:"sender_id"`
	SentAt    string `json:"sent_at"`
	IsMine    bool   `json:"is_mine"`
}

type PaginationInfo struct {
	Limit   int32 `json:"limit"`
	Offset  int32 `json:"offset"`
	HasMore bool  `json:"has_more"`
	Total   int32 `json:"total"`
}

type GetMessagesResponse struct {
	Messages   []MessageInfo  `json:"messages"`
	Pagination PaginationInfo `json:"pagination"`
}

type ConversationInfo struct {
	ConversationID string          `json:"conversation_id"`
	Participant    ParticipantInfo `json:"participant"`
	LastMessage    *LastMessage    `json:"last_message,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

type LastMessage struct {
	Text   string `json:"text"`
	SentAt string `json:"sent_at"`
	IsMine bool   `json:"is_mine"`
}

type GetConversationsResponse struct {
	Conversations []ConversationInfo `json:"conversations"`
	Pagination    PaginationInfo     `json:"pagination"`
}

func (h *Handler) CreateConversation(c *gin.Context) {
	h.logger.Info("CreateConversation endpoint called")

	// Get current user ID from auth middleware
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse request
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate recipient ID
	if req.RecipientID == 0 {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid recipient ID", nil))
		return
	}

	currentUserID := userID.(string)

	// Resolve recipient public ID to UUID using User Service
	recipientUUID, err := h.userClient.ResolveUserID(c.Request.Context(), req.RecipientID)
	if err != nil {
		h.logger.Error("Failed to resolve recipient ID", "error", err, "recipientID", req.RecipientID)
		pkghttp.Error(c, pkghttp.NewNotFound("User not found", nil))
		return
	}

	// Prevent creating conversation with oneself
	if currentUserID == recipientUUID {
		pkghttp.Error(c, pkghttp.NewBadRequest("Cannot create conversation with yourself", nil))
		return
	}

	// Create conversation
	conversation, err := h.chatClient.CreateConversation(c.Request.Context(), currentUserID, recipientUUID)
	if err != nil {
		h.logger.Error("Failed to create conversation", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Get recipient's basic profile information
	recipientProfile, err := h.userClient.GetBasicProfile(c.Request.Context(), recipientUUID)
	if err != nil {
		h.logger.Error("Failed to get recipient profile", "error", err)
		// Don't fail the request, just use basic info
		recipientProfile = &userpb.BasicProfileData{
			Id:       req.RecipientID,
			FullName: "Unknown User",
			IsActive: true,
		}
	}

	// Build response
	response := CreateConversationResponse{
		ConversationID: conversation.Id,
		Participant: ParticipantInfo{
			ID:                recipientProfile.Id,
			Name:              recipientProfile.FullName,
			ProfilePictureURL: nil,
		},
		CreatedAt:     conversation.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
		LastMessageAt: nil,
	}

	if recipientProfile.ProfilePictureUrl != "" {
		response.Participant.ProfilePictureURL = &recipientProfile.ProfilePictureUrl
	}

	if conversation.LastMessageAt != nil {
		lastMsgAt := conversation.LastMessageAt.AsTime().Format("2006-01-02T15:04:05Z07:00")
		response.LastMessageAt = &lastMsgAt
	}

	pkghttp.Success(c, http.StatusOK, "Conversation created successfully", response)
}
func (h *Handler) SendMessage(c *gin.Context) {
	h.logger.Info("SendMessage endpoint called")
	conversationID := c.Param("id")
	if conversationID == "" {
		h.logger.Error("Missing conversation ID in URL")
		pkghttp.Error(c, pkghttp.NewBadRequest("Missing conversation ID", nil))
		return
	}

	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	if len(req.Text) == 0 {
		pkghttp.Error(c, pkghttp.NewBadRequest("Message text cannot be empty", nil))
		return
	}

	if len(req.Text) > 2000 {
		pkghttp.Error(c, pkghttp.NewBadRequest("Message text cannot exceed 2000 characters", nil))
		return
	}

	currentUserID := userID.(string)
	message, err := h.chatClient.SendMessage(c.Request.Context(), currentUserID, conversationID, req.Text)
	if err != nil {
		h.logger.Error("Failed to send message", "error", err, "conversationID", conversationID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Broadcast via WebSocket
	wsMessage := &Message{
		Type:           MessageTypeMessage,
		ConversationID: conversationID,
		MessageID:      message.Id,
		SenderID:       message.SenderId,
		Text:           message.Text,
		SentAt:         message.SentAt.AsTime(),
	}
	h.hub.BroadcastToConversation(conversationID, wsMessage)

	response := gin.H{
		"message_id": message.Id,
		"text":       message.Text,
		"sender_id":  message.SenderId,
		"sent_at":    message.SentAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}
	pkghttp.Success(c, http.StatusCreated, "Message sent successfully", response)
}

func (h *Handler) GetMessages(c *gin.Context) {
	h.logger.Info("GetMessages endpoint called")

	// Get conversation ID from URL parameter
	conversationID := c.Param("id")
	if conversationID == "" {
		h.logger.Error("Missing conversation ID in URL")
		pkghttp.Error(c, pkghttp.NewBadRequest("Missing conversation ID", nil))
		return
	}

	// Get current user ID from auth middleware
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	currentUserID := userID.(string)

	// Get messages via chat service
	messages, pagination, err := h.chatClient.GetMessages(c.Request.Context(), currentUserID, conversationID, int32(limit), int32(offset))
	if err != nil {
		h.logger.Error("Failed to get messages", "error", err, "conversationID", conversationID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Build response
	var messageInfos []MessageInfo
	for _, message := range messages {
		messageInfo := MessageInfo{
			MessageID: message.Id,
			Text:      message.Text,
			SenderID:  message.SenderId,
			SentAt:    message.SentAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
			IsMine:    message.SenderId == currentUserID,
		}
		messageInfos = append(messageInfos, messageInfo)
	}

	response := GetMessagesResponse{
		Messages: messageInfos,
		Pagination: PaginationInfo{
			Limit:   pagination.Limit,
			Offset:  pagination.Offset,
			HasMore: pagination.HasMore,
			Total:   pagination.Total,
		},
	}

	pkghttp.Success(c, http.StatusOK, "Messages retrieved successfully", response)
}

func (h *Handler) GetConversations(c *gin.Context) {
	h.logger.Info("GetConversations endpoint called")

	// Get current user ID from auth middleware
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	currentUserID := userID.(string)

	// Get conversations via chat service
	conversations, pagination, err := h.chatClient.GetUserConversations(c.Request.Context(), currentUserID, int32(limit), int32(offset))
	if err != nil {
		h.logger.Error("Failed to get conversations", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Build response with participant resolution
	var conversationInfos []ConversationInfo
	for _, conversation := range conversations {
		// For MVP, we assume there's only one other participant (1-to-1 chat)
		if len(conversation.Participants) == 0 {
			continue // Skip conversations with no other participants
		}

		otherUserUUID := conversation.Participants[0] // First participant is the other user

		// Resolve participant UUID to public profile via User Service
		participantProfile, err := h.userClient.GetBasicProfile(c.Request.Context(), otherUserUUID)
		if err != nil {
			h.logger.Error("Failed to get participant profile", "error", err, "participantUUID", otherUserUUID)
			// Don't fail the request, use fallback data
			participantProfile = &userpb.BasicProfileData{
				Id:       0, // Will be resolved later or set to default
				FullName: "Unknown User",
				IsActive: true,
			}
		}

		conversationInfo := ConversationInfo{
			ConversationID: conversation.Id,
			Participant: ParticipantInfo{
				ID:                participantProfile.Id,
				Name:              participantProfile.FullName,
				ProfilePictureURL: nil,
			},
			CreatedAt: conversation.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
		}

		if participantProfile.ProfilePictureUrl != "" {
			conversationInfo.Participant.ProfilePictureURL = &participantProfile.ProfilePictureUrl
		}

		// Add last message info if available
		if conversation.LastMessage != nil {
			conversationInfo.LastMessage = &LastMessage{
				Text:   conversation.LastMessage.Text,
				SentAt: conversation.LastMessage.SentAt.AsTime().Format("2006-01-02T15:04:05Z07:00"),
				IsMine: conversation.LastMessage.SenderId == currentUserID,
			}
		}

		conversationInfos = append(conversationInfos, conversationInfo)
	}

	response := GetConversationsResponse{
		Conversations: conversationInfos,
		Pagination: PaginationInfo{
			Limit:   pagination.Limit,
			Offset:  pagination.Offset,
			HasMore: pagination.HasMore,
			Total:   pagination.Total,
		},
	}

	pkghttp.Success(c, http.StatusOK, "Conversations retrieved successfully", response)
}

// TODO: Implement chat endpoints in Phase 2

// func (h *Handler) DeleteMessage(c *gin.Context) {
//     // Implementation will be added in Phase 2
//     pkghttp.Error(c, pkghttp.NewNotImplemented("Chat feature coming soon", nil))
// }

// Placeholder endpoint for Phase 1 - just returns a message
func (h *Handler) ChatStatus(c *gin.Context) {
	h.logger.Info("Chat status endpoint called")

	pkghttp.Success(c, http.StatusOK, "Chat service is ready for Phase 2 implementation", gin.H{
		"status":  "ready",
		"service": "chat",
		"phase":   "1 - foundation complete",
		"next":    "Phase 2 - implement chat functionality",
	})
}
