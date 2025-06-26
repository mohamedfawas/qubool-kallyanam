package admin

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	adminpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/admin/v1"
)

// Client wraps the admin service client
type Client struct {
	conn   *grpc.ClientConn
	client adminpb.AdminServiceClient
}

// NewClient creates and returns a new gRPC client to communicate with the Admin service
func NewClient(address string) (*Client, error) {
	clientConn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		conn:   clientConn,
		client: adminpb.NewAdminServiceClient(clientConn),
	}, nil
}

// GetUsers calls the admin service to get list of users with filtering
func (c *Client) GetUsers(ctx context.Context, req *adminpb.GetUsersRequest) (*adminpb.GetUsersResponse, error) {
	return c.client.GetUsers(ctx, req)
}

// GetUser calls the admin service to get detailed user information
func (c *Client) GetUser(ctx context.Context, req *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	return c.client.GetUser(ctx, req)
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}
