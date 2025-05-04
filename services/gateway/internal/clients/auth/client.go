package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
)

// Client wraps the auth service client
type Client struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

// NewClient creates a new auth service client
func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: authpb.NewAuthServiceClient(conn),
	}, nil
}

// Register sends a registration request to the auth service
func (c *Client) Register(ctx context.Context, email, phone, password string) (bool, string, error) {
	resp, err := c.client.Register(ctx, &authpb.RegisterRequest{
		Email:    email,
		Phone:    phone,
		Password: password,
	})
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}
