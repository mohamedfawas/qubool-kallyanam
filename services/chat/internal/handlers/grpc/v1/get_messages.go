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

func (h *ChatHandler) GetMessages(ctx context.Context, req *chatpb.GetMessagesRequest) (*chatpb.GetMessagesResponse, error) {
	h.logger.Info("GetMessages gRPC request", "userID", req.UserId, "conversationID", req.ConversationId, "limit", req.Limit, "offset", req.Offset)

	// Validate request
	if req.UserId == "" || req.ConversationId == "" {
		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID and Conversation ID are required",
		}, status.Error(codes.InvalidArgument, "User ID and Conversation ID are required")
	}

	// Parse conversation ID
	conversationID, err := primitive.ObjectIDFromHex(req.ConversationId)
	if err != nil {
		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Invalid conversation ID",
			Error:   "Invalid conversation ID format",
		}, status.Error(codes.InvalidArgument, "Invalid conversation ID format")
	}

	// Set defaults for pagination
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	// Call service
	messages, total, err := h.chatService.GetMessages(ctx, req.UserId, conversationID, int(limit), int(offset))
	if err != nil {
		h.logger.Error("Failed to get messages", "error", err)

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
			errMsg = "Not authorized to view messages in this conversation"
			statusCode = codes.PermissionDenied
		case errors.Is(err, mongo.ErrNoDocuments):
			errMsg = "Conversation not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Failed to get messages"
			statusCode = codes.Internal
		}

		return &chatpb.GetMessagesResponse{
			Success: false,
			Message: "Failed to get messages",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build response
	var messageDataList []*chatpb.MessageData
	for _, message := range messages {
		messageData := &chatpb.MessageData{
			Id:             message.ID.Hex(),
			ConversationId: message.ConversationID.Hex(),
			SenderId:       message.SenderID,
			Text:           message.Text,
			SentAt:         timestamppb.New(message.SentAt),
		}
		messageDataList = append(messageDataList, messageData)
	}

	// Calculate has_more
	hasMore := int32(offset)+limit < int32(total)

	pagination := &chatpb.PaginationData{
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
		Total:   int32(total),
	}

	return &chatpb.GetMessagesResponse{
		Success:    true,
		Message:    "Messages retrieved successfully",
		Messages:   messageDataList,
		Pagination: pagination,
	}, nil
}
