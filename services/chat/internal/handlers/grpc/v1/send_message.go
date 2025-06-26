package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/helpers"
)

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatpb.SendMessageRequest) (*chatpb.SendMessageResponse, error) {
	h.logger.Info("SendMessage gRPC request", "userID", req.UserId, "conversationID", req.ConversationId)

	// Validate request using helper
	if err := helpers.ValidateSendMessageInput(req.UserId, req.ConversationId, req.Text); err != nil {
		h.logger.Debug("Invalid send message request", "error", err)
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Parse conversation ID using helper
	conversationID, err := helpers.ValidateObjectID(req.ConversationId)
	if err != nil {
		h.logger.Debug("Invalid conversation ID format", "conversationID", req.ConversationId)
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Invalid conversation ID",
			Error:   "Invalid conversation ID format",
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Call service
	message, err := h.chatService.SendMessage(ctx, req.UserId, conversationID, req.Text)
	if err != nil {
		h.logger.Error("Failed to send message", "error", err)
		return &chatpb.SendMessageResponse{
			Success: false,
			Message: "Failed to send message",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
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
