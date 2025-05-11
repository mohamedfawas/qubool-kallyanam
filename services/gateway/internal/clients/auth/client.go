package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
)

// Client wraps the auth service client
type Client struct {
	conn   *grpc.ClientConn         // Connection to the auth service
	client authpb.AuthServiceClient // AuthServiceClient generated from proto file
}

// NewClient creates and returns a new gRPC client to communicate with the Auth service.
// It establishes a connection to the gRPC server at the given address.
func NewClient(address string) (*Client, error) {
	// Step 1: Create a new gRPC client connection to the provided server address.
	// The connection uses insecure credentials (no TLS), which is fine for development
	// or trusted internal environments. In production, use proper TLS credentials.
	clientConn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // disables TLS encryption
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Step 3: Initialize and return the custom Client struct.
	// - `conn` holds the low-level gRPC connection object.
	// - `client` is the generated AuthServiceClient which provides methods
	//   to call the remote RPC endpoints defined in the proto file.
	return &Client{
		conn:   clientConn,
		client: authpb.NewAuthServiceClient(clientConn),
	}, nil
}

// Register sends a registration request to the auth service
func (c *Client) Register(ctx context.Context,
	email, phone, password string) (bool, string, error) {

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

// Verify sends a verification request to the auth service
func (c *Client) Verify(ctx context.Context, email, otp string) (bool, string, error) {
	resp, err := c.client.Verify(ctx, &authpb.VerifyRequest{
		Email: email,
		Otp:   otp,
	})
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

// Login sends authentication request to the auth service
func (c *Client) Login(ctx context.Context, email, password string) (bool, string, string, string, int32, error) {
	resp, err := c.client.Login(ctx, &authpb.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return false, "", "", "", 0, err
	}

	return resp.Success, resp.AccessToken, resp.RefreshToken, resp.Message, resp.ExpiresIn, nil
}

// Logout sends a logout request to the auth service
func (c *Client) Logout(ctx context.Context, accessToken string) (bool, string, error) {
	resp, err := c.client.Logout(ctx, &authpb.LogoutRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		return false, "", err
	}

	return resp.Success, resp.Message, nil
}

// RefreshToken sends a token refresh request to the auth service
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (bool, string, string, int32, string, error) {
	// Create metadata with the authorization header
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + refreshToken,
	})

	// Create new context with the metadata
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Call the service with empty request
	resp, err := c.client.RefreshToken(ctx, &authpb.RefreshTokenRequest{})
	if err != nil {
		return false, "", "", 0, "", err
	}

	return resp.Success, resp.AccessToken, resp.RefreshToken, resp.ExpiresIn, resp.Message, nil
}

// AdminLogin sends admin authentication request to the auth service
func (c *Client) AdminLogin(ctx context.Context, email, password string) (bool, string, string, string, int32, error) {
	resp, err := c.client.AdminLogin(ctx, &authpb.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return false, "", "", "", 0, err
	}

	return resp.Success, resp.AccessToken, resp.RefreshToken, resp.Message, resp.ExpiresIn, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}
