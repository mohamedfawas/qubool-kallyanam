package server

import (
	"context"
	"log"

	authv1 "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"gorm.io/gorm"
)

// AuthServer implements the auth gRPC service
type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	db          *gorm.DB
	redisClient *redis.Client
}

// NewAuthServer creates a new auth server
func NewAuthServer(db *gorm.DB, redisClient *redis.Client) *AuthServer {
	return &AuthServer{
		db:          db,
		redisClient: redisClient,
	}
}

// HealthCheck implements the HealthCheck RPC method
func (s *AuthServer) HealthCheck(ctx context.Context, req *authv1.HealthCheckRequest) (*authv1.HealthCheckResponse, error) {
	log.Println("Health check requested")

	status := "UP"

	// Check database connection
	sqlDB, err := s.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		log.Printf("Postgres health check failed: %v", err)
		status = "DOWN"
	}

	// Check Redis connection
	_, err = s.redisClient.Get(ctx, "health_check_key")
	if err != nil && err.Error() != "redis: nil" {
		log.Printf("Redis health check failed: %v", err)
		status = "DOWN"
	}

	return &authv1.HealthCheckResponse{
		Status: status,
	}, nil
}

// RegisterUser implements the RegisterUser RPC method
func (s *AuthServer) RegisterUser(ctx context.Context, req *authv1.RegisterUserRequest) (*authv1.RegisterUserResponse, error) {
	log.Printf("Received registration request for email: %s, phone: %s", req.Email, req.Phone)

	// For MVP, just return success
	return &authv1.RegisterUserResponse{
		Success: true,
		Message: "Registration initiated successfully",
	}, nil
}

// Login implements the Login RPC method
func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	log.Printf("Login attempt for identifier: %s", req.Identifier)

	// For MVP, just return success
	return &authv1.LoginResponse{
		Success:      true,
		Message:      "Login successful",
		AccessToken:  "sample-access-token",
		RefreshToken: "sample-refresh-token",
	}, nil
}

// VerifyRegistration implements the VerifyRegistration RPC method
func (s *AuthServer) VerifyRegistration(ctx context.Context, req *authv1.VerifyRegistrationRequest) (*authv1.VerifyRegistrationResponse, error) {
	log.Printf("Verification attempt for email: %s, OTP: %s", req.Email, req.Otp)

	// For MVP, just return success
	return &authv1.VerifyRegistrationResponse{
		Success:      true,
		Message:      "Verification successful",
		AccessToken:  "sample-access-token",
		RefreshToken: "sample-refresh-token",
	}, nil
}
