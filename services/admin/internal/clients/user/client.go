package user

import (
	"context"
	"fmt"
	"time"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
}

func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: userpb.NewUserServiceClient(conn),
	}, nil
}

// GetProfileForAdmin gets detailed profile for admin view
func (c *Client) GetProfileForAdmin(ctx context.Context, req *userpb.GetProfileForAdminRequest) (*userpb.GetDetailedProfileResponse, error) {
	return c.client.GetProfileForAdmin(ctx, req)
}

func (c *Client) Close() error {
	return c.conn.Close()
}
