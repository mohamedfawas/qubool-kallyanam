package v1

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/services"
)

// AuthHandler implements the auth gRPC service
type AuthHandler struct {
	authpb.UnimplementedAuthServiceServer
	registrationService *services.RegistrationService
	authService         *services.AuthService
	logger              logging.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	registrationService *services.RegistrationService,
	authService *services.AuthService,
	logger logging.Logger,
) *AuthHandler {
	return &AuthHandler{
		registrationService: registrationService,
		authService:         authService,
		logger:              logger,
	}
}

// Register handles user registration requests
func (h *AuthHandler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	h.logger.Info("Received registration request", "email", req.Email, "phone", req.Phone)

	// Create registration model from request
	registration := &models.Registration{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	// Process registration
	err := h.registrationService.RegisterUser(ctx, registration)
	if err != nil {
		h.logger.Error("Registration failed", "error", err)

		// Map domain errors to gRPC status codes
		switch {
		case err == services.ErrEmailAlreadyExists:
			return nil, status.Error(codes.AlreadyExists, "Email already registered")
		case err == services.ErrPhoneAlreadyExists:
			return nil, status.Error(codes.AlreadyExists, "Phone number already registered")
		case err == services.ErrOTPGenerationFailed:
			return nil, status.Error(codes.Internal, "Failed to generate verification code")
		default:
			if services.ErrInvalidInput.Error() == err.Error() {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			return nil, status.Error(codes.Internal, "Registration failed")
		}
	}

	h.logger.Info("Registration successful, OTP sent", "email", req.Email)

	// Return successful response
	return &authpb.RegisterResponse{
		Success: true,
		Message: "OTP sent to registered email",
	}, nil
}

// Verify handles user verification with OTP
func (h *AuthHandler) Verify(ctx context.Context, req *authpb.VerifyRequest) (*authpb.VerifyResponse, error) {
	h.logger.Info("Received verification request", "email", req.Email)

	// Call service to verify the registration
	err := h.registrationService.VerifyRegistration(ctx, req.Email, req.Otp)
	if err != nil {
		h.logger.Error("Verification failed", "error", err)

		// Map domain errors to appropriate gRPC status codes
		switch {
		case err == services.ErrInvalidOTP:
			return nil, status.Error(codes.InvalidArgument, "Invalid or expired OTP")
		case strings.Contains(err.Error(), "no pending registration found"):
			return nil, status.Error(codes.NotFound, "No pending registration found")
		case services.ErrInvalidInput.Error() == err.Error():
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "Verification failed")
		}
	}

	h.logger.Info("Verification successful", "email", req.Email)

	// Return success response
	return &authpb.VerifyResponse{
		Success: true,
		Message: "Account verified successfully",
	}, nil
}

// Login handles user authentication and token generation
func (h *AuthHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	h.logger.Info("Received login request", "email", req.Email)

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.logger.Debug("Invalid login request - missing required fields")
		return nil, status.Error(codes.InvalidArgument, "Email and password are required")
	}

	// Call authentication service to process login
	tokenPair, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		// Map domain errors to appropriate gRPC status codes
		switch {
		case err == services.ErrInvalidCredentials:
			h.logger.Debug("Login failed - invalid credentials", "email", req.Email)
			return nil, status.Error(codes.Unauthenticated, "Invalid email or password")
		case err == services.ErrAccountNotVerified:
			h.logger.Debug("Login failed - account not verified", "email", req.Email)
			return nil, status.Error(codes.PermissionDenied, "Account not verified")
		case err == services.ErrAccountDisabled:
			h.logger.Debug("Login failed - account disabled", "email", req.Email)
			return nil, status.Error(codes.PermissionDenied, "Account is disabled")
		default:
			h.logger.Error("Login failed - internal error", "email", req.Email, "error", err)
			return nil, status.Error(codes.Internal, "Authentication failed")
		}
	}

	h.logger.Info("Login successful", "email", req.Email)

	// Return successful response with tokens
	return &authpb.LoginResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Login successful",
	}, nil
}

// Logout handles user logout requests
func (h *AuthHandler) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	h.logger.Info("Received logout request")

	if req.AccessToken == "" {
		h.logger.Debug("Invalid logout request - missing token")
		return nil, status.Error(codes.InvalidArgument, "Access token is required")
	}

	// Call authentication service to process logout
	err := h.authService.Logout(ctx, req.AccessToken)
	if err != nil {
		// Map domain errors to appropriate gRPC status codes
		switch {
		case err == services.ErrInvalidToken:
			h.logger.Debug("Logout failed - invalid token")
			return nil, status.Error(codes.Unauthenticated, "Invalid or expired token")
		default:
			h.logger.Error("Logout failed - internal error", "error", err)
			return nil, status.Error(codes.Internal, "Logout failed")
		}
	}

	h.logger.Info("Logout successful")

	// Return successful response
	return &authpb.LogoutResponse{
		Success: true,
		Message: "Logout successful",
	}, nil
}

// RefreshToken handles token refresh requests
func (h *AuthHandler) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	h.logger.Info("Received token refresh request")

	// Extract token from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		h.logger.Debug("Metadata missing from context")
		return nil, status.Error(codes.Unauthenticated, "Missing authorization")
	}

	// Get authorization values
	values := md.Get("authorization")
	if len(values) == 0 {
		h.logger.Debug("Authorization header missing")
		return nil, status.Error(codes.Unauthenticated, "Missing authorization header")
	}

	// Extract token from "Bearer token" format
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		h.logger.Debug("Invalid authorization format")
		return nil, status.Error(codes.Unauthenticated, "Invalid authorization format")
	}

	refreshToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Call service to refresh tokens
	tokenPair, err := h.authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		// Map domain errors to appropriate gRPC status codes (unchanged)
		switch {
		case err == services.ErrInvalidRefreshToken:
			h.logger.Debug("Refresh failed - invalid token")
			return nil, status.Error(codes.Unauthenticated, "Invalid or expired refresh token")
		default:
			h.logger.Error("Refresh failed - internal error", "error", err)
			return nil, status.Error(codes.Internal, "Token refresh failed")
		}
	}

	h.logger.Info("Token refresh successful")

	// Return successful response with new tokens
	return &authpb.RefreshTokenResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Token refresh successful",
	}, nil
}

func (h *AuthHandler) AdminLogin(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	h.logger.Info("Received admin login request", "email", req.Email)

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.logger.Debug("Invalid admin login request - missing required fields")
		return nil, status.Error(codes.InvalidArgument, "Email and password are required")
	}

	// Call authentication service to process admin login
	tokenPair, err := h.authService.AdminLogin(ctx, req.Email, req.Password)
	if err != nil {
		// Map domain errors to appropriate gRPC status codes
		switch {
		case err == services.ErrAdminNotFound || err == services.ErrInvalidCredentials:
			h.logger.Debug("Admin login failed - invalid credentials", "email", req.Email)
			return nil, status.Error(codes.Unauthenticated, "Invalid email or password")
		case err == services.ErrAdminAccountDisabled:
			h.logger.Debug("Admin login failed - account disabled", "email", req.Email)
			return nil, status.Error(codes.PermissionDenied, "Admin account is disabled")
		default:
			h.logger.Error("Admin login failed - internal error", "email", req.Email, "error", err)
			return nil, status.Error(codes.Internal, "Authentication failed")
		}
	}

	h.logger.Info("Admin login successful", "email", req.Email)

	// Return successful response with tokens
	return &authpb.LoginResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Admin login successful",
	}, nil
}
