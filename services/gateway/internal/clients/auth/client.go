package auth

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
)

// Client is a client for the auth service
type Client struct {
	conn   *grpc.ClientConn
	client authv1.AuthServiceClient
}

// NewClient creates a new auth service client
func NewClient(address string) (*Client, error) {
	fmt.Printf("Connecting to auth service at: %s\n", address)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &Client{
		conn:   conn,
		client: authv1.NewAuthServiceClient(conn),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// HealthCheck checks the health of the auth service
func (c *Client) HealthCheck(ctx context.Context) (string, error) {
	resp, err := c.client.HealthCheck(ctx, &authv1.HealthCheckRequest{})
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}

// RegisterUser registers a new user
func (c *Client) RegisterUser(ctx context.Context, email, phone, password string) (*authv1.RegisterUserResponse, error) {
	return c.client.RegisterUser(ctx, &authv1.RegisterUserRequest{
		Email:    email,
		Phone:    phone,
		Password: password,
	})
}

// Login attempts to log in a user
func (c *Client) Login(ctx context.Context, identifier, password string) (*authv1.LoginResponse, error) {
	return c.client.Login(ctx, &authv1.LoginRequest{
		Identifier: identifier,
		Password:   password,
	})
}

// VerifyRegistration verifies a registration OTP
func (c *Client) VerifyRegistration(ctx context.Context, email, otp string) (*authv1.VerifyRegistrationResponse, error) {
	return c.client.VerifyRegistration(ctx, &authv1.VerifyRegistrationRequest{
		Email: email,
		Otp:   otp,
	})
}
