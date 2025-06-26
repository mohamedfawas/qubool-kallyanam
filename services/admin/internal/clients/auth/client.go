package auth

import (
	"context"
	"fmt"
	"time"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: authpb.NewAuthServiceClient(conn),
	}, nil
}

func (c *Client) GetUsersList(ctx context.Context, req *authpb.GetUsersListRequest) (*authpb.GetUsersListResponse, error) {
	return c.client.GetUsersList(ctx, req)
}

func (c *Client) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.GetUserResponse, error) {
	return c.client.GetUser(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
