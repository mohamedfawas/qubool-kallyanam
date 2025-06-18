package helpers

import (
	autherrors "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapErrorToGRPCStatus maps internal errors to gRPC status codes
func MapErrorToGRPCStatus(err error) error {
	switch err {
	// Common errors
	case autherrors.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, err.Error())

	// User errors
	case autherrors.ErrUserNotFound:
		return status.Error(codes.NotFound, err.Error())
	case autherrors.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, "Invalid email or password")
	case autherrors.ErrAccountNotVerified:
		return status.Error(codes.PermissionDenied, "Account not verified")
	case autherrors.ErrAccountDisabled:
		return status.Error(codes.PermissionDenied, "Account is disabled")
	case autherrors.ErrEmailAlreadyExists:
		return status.Error(codes.AlreadyExists, "Email already registered")
	case autherrors.ErrPhoneAlreadyExists:
		return status.Error(codes.AlreadyExists, "Phone number already registered")
	case autherrors.ErrRegistrationFailed:
		return status.Error(codes.Internal, "Registration failed")
	case autherrors.ErrVerificationFailed:
		return status.Error(codes.Internal, "Verification failed")

	// Token errors
	case autherrors.ErrInvalidToken:
		return status.Error(codes.Unauthenticated, "Invalid or expired token")
	case autherrors.ErrInvalidRefreshToken:
		return status.Error(codes.Unauthenticated, "Invalid or expired refresh token")

	// OTP errors
	case autherrors.ErrInvalidOTP:
		return status.Error(codes.InvalidArgument, "Invalid or expired OTP")
	case autherrors.ErrOTPGenerationFailed:
		return status.Error(codes.Internal, "Failed to generate verification code")

	// Admin errors
	case autherrors.ErrAdminNotFound:
		return status.Error(codes.Unauthenticated, "Invalid email or password")
	case autherrors.ErrAdminAccountDisabled:
		return status.Error(codes.PermissionDenied, "Admin account is disabled")
	case autherrors.ErrInvalidAdminInput:
		return status.Error(codes.InvalidArgument, err.Error())

	// Default case
	default:
		return status.Error(codes.Internal, "Internal server error")
	}
}
