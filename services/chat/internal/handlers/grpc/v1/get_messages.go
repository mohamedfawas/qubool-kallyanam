package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	chaterrors "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/errors"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/helpers"
)

func (h *ChatHandler) GetMessages(ctx context.Context, req *chatpb.GetMessagesRequest) (*chatpb.GetMessagesResponse, error) {
	h.logger.Info("GetMessages gRPC request", "userID", req.UserId, "conversationID", req.ConversationId, "limit", req.Limit, "offset", req.Offset)

	// Validate input
	if req.UserId == "" || req.ConversationId == "" {
		h.logger.Debug("Invalid get messages request - missing required fields")
		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID and Conversation ID are required",
		}, helpers.MapErrorToGRPCStatus(chaterrors.ErrInvalidInput)
	}

	// Parse conversation ID using helper
	conversationID, err := helpers.ValidateObjectID(req.ConversationId)
	if err != nil {
		h.logger.Debug("Invalid conversation ID format", "conversationID", req.ConversationId)
		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Invalid conversation ID",
			Error:   "Invalid conversation ID format",
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Validate and normalize pagination using helper
	limit, offset := helpers.ValidatePaginationParams(
		int(req.Limit),
		int(req.Offset),
		constants.MaxMessageLimit,
		constants.DefaultMessageLimit,
	)

	// Call service
	messages, total, err := h.chatService.GetMessages(ctx, req.UserId, conversationID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get messages", "error", err)
		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Failed to get messages",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Build response
	messageData := make([]*chatpb.MessageData, len(messages))
	for i, message := range messages {
		messageData[i] = &chatpb.MessageData{
			Id:             message.ID.Hex(),
			ConversationId: message.ConversationID.Hex(),
			SenderId:       message.SenderID,
			Text:           message.Text,
			SentAt:         timestamppb.New(message.SentAt),
		}
	}

	// Check if there are more messages
	hasMore := len(messages) == limit && (offset+limit) < total

	return &chatpb.GetMessagesResponse{
		Success:  true,
		Message:  "Messages retrieved successfully",
		Messages: messageData,
		Pagination: &chatpb.PaginationData{
			Limit:   int32(limit),
			Offset:  int32(offset),
			HasMore: hasMore,
			Total:   int32(total),
		},
	}, nil
}
