package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	chaterrors "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/errors"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/helpers"
)

func (h *ChatHandler) GetUserConversations(ctx context.Context, req *chatpb.GetUserConversationsRequest) (*chatpb.GetUserConversationsResponse, error) {
	h.logger.Info("GetUserConversations gRPC request", "userID", req.UserId, "limit", req.Limit, "offset", req.Offset)

	// Validate input
	if req.UserId == "" {
		h.logger.Debug("Invalid get user conversations request - missing user ID")
		return &chatpb.GetUserConversationsResponse{
			Success: false,
			Message: "Invalid request parameters",
			Error:   "User ID is required",
		}, helpers.MapErrorToGRPCStatus(chaterrors.ErrInvalidInput)
	}

	// Validate and normalize pagination using helper
	limit, offset := helpers.ValidatePaginationParams(
		int(req.Limit),
		int(req.Offset),
		constants.MaxConversationLimit,
		constants.DefaultConversationLimit,
	)

	// Call service
	conversations, total, err := h.chatService.GetUserConversations(ctx, req.UserId, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user conversations", "error", err)
		return &chatpb.GetUserConversationsResponse{
			Success: false,
			Message: "Failed to get conversations",
			Error:   err.Error(),
		}, helpers.MapErrorToGRPCStatus(err)
	}

	// Build response using ConversationSummary
	conversationSummaries := make([]*chatpb.ConversationSummary, len(conversations))
	for i, conversation := range conversations {
		// Filter participants to exclude current user for summary
		otherParticipants := make([]string, 0, len(conversation.Participants)-1)
		for _, participant := range conversation.Participants {
			if participant != req.UserId {
				otherParticipants = append(otherParticipants, participant)
			}
		}

		conversationSummary := &chatpb.ConversationSummary{
			Id:           conversation.ID.Hex(),
			Participants: otherParticipants, // Only other participants, excluding current user
			CreatedAt:    timestamppb.New(conversation.CreatedAt),
		}

		if conversation.LastMessageAt != nil {
			conversationSummary.LastMessageAt = timestamppb.New(*conversation.LastMessageAt)
		}

		// Get latest message for this conversation
		if latestMessage, err := h.chatService.GetLatestMessage(context.Background(), conversation.ID); err == nil {
			conversationSummary.LastMessage = &chatpb.LastMessageData{
				Text:     latestMessage.Text,
				SenderId: latestMessage.SenderID,
				SentAt:   timestamppb.New(latestMessage.SentAt),
			}
		}

		conversationSummaries[i] = conversationSummary
	}

	// Check if there are more conversations
	hasMore := len(conversations) == limit && (offset+limit) < total

	return &chatpb.GetUserConversationsResponse{
		Success:       true,
		Message:       "Conversations retrieved successfully",
		Conversations: conversationSummaries,
		Pagination: &chatpb.PaginationData{
			Limit:   int32(limit),
			Offset:  int32(offset),
			HasMore: hasMore,
			Total:   int32(total),
		},
	}, nil
}
