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

func (h *ChatHandler) GetUserConversations(ctx context.Context, req *chatpb.GetUserConversationsRequest) (*chatpb.GetUserConversationsResponse, error) {
	h.logger.Info("GetUserConversations gRPC request", "userID", req.UserId, "limit", req.Limit, "offset", req.Offset)

	// Validate request
	if req.UserId == "" {
		return &chatpb.GetUserConversationsResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID is required",
		}, status.Error(codes.InvalidArgument, "User ID is required")
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
	conversations, total, err := h.chatService.GetUserConversations(ctx, req.UserId, int(limit), int(offset))
	if err != nil {
		h.logger.Error("Failed to get user conversations", "error", err)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = "Invalid input parameters"
			statusCode = codes.InvalidArgument
		case errors.Is(err, mongo.ErrNoDocuments):
			errMsg = "No conversations found"
			statusCode = codes.NotFound
		default:
			errMsg = "Failed to get conversations"
			statusCode = codes.Internal
		}

		return &chatpb.GetUserConversationsResponse{
			Success: false,
			Message: "Failed to get conversations",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build response
	var conversationSummaries []*chatpb.ConversationSummary
	for _, conversation := range conversations {
		// Filter out current user from participants list
		var otherParticipants []string
		for _, participant := range conversation.Participants {
			if participant != req.UserId {
				otherParticipants = append(otherParticipants, participant)
			}
		}

		conversationSummary := &chatpb.ConversationSummary{
			Id:           conversation.ID.Hex(),
			Participants: otherParticipants,
			CreatedAt:    timestamppb.New(conversation.CreatedAt),
		}

		if conversation.LastMessageAt != nil {
			conversationSummary.LastMessageAt = timestamppb.New(*conversation.LastMessageAt)
		}

		// Get latest message for this conversation (optional for MVP - can be nil)
		if conversation.LastMessageAt != nil {
			latestMessage, err := h.chatService.GetLatestMessage(ctx, conversation.ID)
			if err == nil && latestMessage != nil {
				conversationSummary.LastMessage = &chatpb.LastMessageData{
					Text:     latestMessage.Text,
					SenderId: latestMessage.SenderID,
					SentAt:   timestamppb.New(latestMessage.SentAt),
				}
			}
		}

		conversationSummaries = append(conversationSummaries, conversationSummary)
	}

	// Calculate has_more
	hasMore := int32(offset)+limit < int32(total)

	pagination := &chatpb.PaginationData{
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
		Total:   int32(total),
	}

	return &chatpb.GetUserConversationsResponse{
		Success:       true,
		Message:       "Conversations retrieved successfully",
		Conversations: conversationSummaries,
		Pagination:    pagination,
	}, nil
}
