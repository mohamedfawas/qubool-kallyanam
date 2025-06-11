package v1

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/services"
)

func (h *ChatHandler) CreateConversation(ctx context.Context, req *chatpb.CreateConversationRequest) (*chatpb.CreateConversationResponse, error) {
	h.logger.Info("CreateConversation gRPC request", "userID", req.UserId, "participantID", req.ParticipantId)

	// Validate request
	if req.UserId == "" || req.ParticipantId == "" {
		return &chatpb.CreateConversationResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID and Participant ID are required",
		}, status.Error(codes.InvalidArgument, "User ID and Participant ID are required")
	}

	// Call service
	conversation, err := h.chatService.CreateConversation(ctx, req.UserId, req.ParticipantId)
	if err != nil {
		h.logger.Error("Failed to create conversation", "error", err)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = "Invalid input parameters"
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrDuplicateParticipant):
			errMsg = "Cannot create conversation with yourself"
			statusCode = codes.InvalidArgument
		case errors.Is(err, mongo.ErrNoDocuments):
			errMsg = "User not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Failed to create conversation"
			statusCode = codes.Internal
		}

		return &chatpb.CreateConversationResponse{
			Success: false,
			Message: "Failed to create conversation",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
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
