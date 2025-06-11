package v1

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/services"
)

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatpb.SendMessageRequest) (*chatpb.SendMessageResponse, error) {
	h.logger.Info("SendMessage gRPC request", "userID", req.UserId, "conversationID", req.ConversationId)

	// Validate request
	if req.UserId == "" || req.ConversationId == "" || req.Text == "" {
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID, Conversation ID and text are required",
		}, status.Error(codes.InvalidArgument, "User ID, Conversation ID and text are required")
	}

	// Validate text length
	if len(req.Text) > 2000 {
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Message too long",
			Error:   "Message text cannot exceed 2000 characters",
		}, status.Error(codes.InvalidArgument, "Message text cannot exceed 2000 characters")
	}

	// Parse conversation ID
	conversationID, err := primitive.ObjectIDFromHex(req.ConversationId)
	if err != nil {
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Invalid conversation ID",
			Error:   "Invalid conversation ID format",
		}, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	// Call service
	message, err := h.chatService.SendMessage(ctx, req.UserId, conversationID, req.Text)
	if err != nil {
		h.logger.Error("Failed to send message", "error", err)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = "Invalid input parameters"
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrConversationNotFound):
			errMsg = "Conversation not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrUnauthorized):
			errMsg = "Not authorized to send message to this conversation"
			statusCode = codes.PermissionDenied
		case errors.Is(err, mongo.ErrNoDocuments):
			errMsg = "Conversation not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Failed to send message"
			statusCode = codes.Internal
		}

		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Failed to send message",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build response
	messageData := &chatpb.MessageData{
		Id:             message.ID.Hex(),
		ConversationId: message.ConversationID.Hex(),
		SenderId:       message.SenderID,
		Text:           message.Text,
		SentAt:         timestamppb.New(message.SentAt),
	}

	return &chatpb.SendMessageResponse{
		Success:     true,
		Message:     "Message sent successfully",
		MessageData: messageData,
	}, nil
}
