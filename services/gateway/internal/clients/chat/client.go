package chat

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
)

type Client struct {
	conn   *grpc.ClientConn
	client chatpb.ChatServiceClient
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: chatpb.NewChatServiceClient(conn),
	}, nil
}

func (c *Client) CreateConversation(ctx context.Context, userID, participantID string) (*chatpb.ConversationData, error) {
	req := &chatpb.CreateConversationRequest{
		UserId:        userID,
		ParticipantId: participantID,
	}

	resp, err := c.client.CreateConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to create conversation: %s", resp.Error)
	}

	return resp.Conversation, nil
}

func (c *Client) SendMessage(ctx context.Context, userID, conversationID, text string) (*chatpb.MessageData, error) {
	req := &chatpb.SendMessageRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Text:           text,
	}

	resp, err := c.client.SendMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("failed to send message: %s", resp.Error)
	}

	return resp.MessageData, nil
}

func (c *Client) GetMessages(ctx context.Context, userID, conversationID string, limit, offset int32) ([]*chatpb.MessageData, *chatpb.PaginationData, error) {
	req := &chatpb.GetMessagesRequest{
		UserId:         userID,
		ConversationId: conversationID,
		Limit:          limit,
		Offset:         offset,
	}

	resp, err := c.client.GetMessages(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if !resp.Success {
		return nil, nil, fmt.Errorf("failed to get messages: %s", resp.Error)
	}

	return resp.Messages, resp.Pagination, nil
}

func (c *Client) GetUserConversations(ctx context.Context, userID string, limit, offset int32) ([]*chatpb.ConversationSummary, *chatpb.PaginationData, error) {
	req := &chatpb.GetUserConversationsRequest{
		UserId: userID,
		Limit:  limit,
		Offset: offset,
	}

	resp, err := c.client.GetUserConversations(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if !resp.Success {
		return nil, nil, fmt.Errorf("failed to get conversations: %s", resp.Error)
	}

	return resp.Conversations, resp.Pagination, nil
}

// TODO: Add chat methods in Phase 2

// func (c *Client) DeleteMessage(ctx context.Context, messageID string) (*chatpb.DeleteMessageResponse, error) {
//     return nil, errors.New("not implemented yet")
// }

func (c *Client) Close() error {
	return c.conn.Close()
}
