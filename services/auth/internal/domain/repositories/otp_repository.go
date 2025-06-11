package repositories

import (
	"context"
	"time"
)

// OTPRepository defines methods for OTP storage operations
type OTPRepository interface {
	// GetOTP retrieves an OTP by key
	GetOTP(ctx context.Context, key string) (string, error)

	// StoreOTP saves an OTP with expiration
	StoreOTP(ctx context.Context, key string, otp string, expiry time.Duration) error

	// DeleteOTP removes an OTP by key
	DeleteOTP(ctx context.Context, key string) error
}
