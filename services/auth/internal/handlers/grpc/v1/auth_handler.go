package v1

import (
	"context"

	"google.golang.org/grpc/codes"
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
	logger              logging.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	registrationService *services.RegistrationService,
	logger logging.Logger,
) *AuthHandler {
	return &AuthHandler{
		registrationService: registrationService,
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
