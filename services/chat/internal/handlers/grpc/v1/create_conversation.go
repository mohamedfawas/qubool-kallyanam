package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/helpers"
)

func (h *ChatHandler) CreateConversation(ctx context.Context, req *chatpb.CreateConversationRequest) (*chatpb.CreateConversationResponse, error) {
	h.logger.Info("CreateConversation gRPC request", "userID", req.UserId, "participantID", req.ParticipantId)

	// Validate request using helper
	if err := helpers.ValidateCreateConversationInput(req.UserId, req.ParticipantId); err != nil {
		h.logger.Debug("Invalid create conversation request", "error", err)
		return &chatpb.CreateConversationResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Call service
	conversation, err := h.chatService.CreateConversation(ctx, req.UserId, req.ParticipantId)
	if err != nil {
		h.logger.Error("Failed to create conversation", "error", err)
		return &chatpb.CreateConversationResponse{
			Success: false,
			Message: "Failed to create conversation",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Build response
	conversationData := &chatpb.ConversationData{
		Id:           conversation.ID.Hex(),
		Participants: conversation.Participants,
		CreatedAt:    timestamppb.New(conversation.CreatedAt),
		UpdatedAt:    timestamppb.New(conversation.UpdatedAt),
	}

	if conversation.LastMessageAt != nil {
		conversationData.LastMessageAt = timestamppb.New(*conversation.LastMessageAt)
	}

	return &chatpb.CreateConversationResponse{
		Success:      true,
		Message:      "Conversation created successfully",
		Conversation: conversationData,
	}, nil
}
