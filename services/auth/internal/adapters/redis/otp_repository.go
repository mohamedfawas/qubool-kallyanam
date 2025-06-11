package redis

import (
	"context"
	"time"

	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

// OTPRepo implements the OTP repository interface using Redis
type OTPRepo struct {
	client *redisdb.Client
}

// NewOTPRepository creates a new OTP repository
func NewOTPRepository(client *redisdb.Client) repositories.OTPRepository {
	return &OTPRepo{
		client: client,
	}
}

// GetOTP retrieves an OTP by key
func (r *OTPRepo) GetOTP(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key)
}

// StoreOTP saves an OTP with expiration
func (r *OTPRepo) StoreOTP(ctx context.Context, key string, otp string, expiry time.Duration) error {
	return r.client.Set(ctx, key, otp, expiry)
}

// DeleteOTP removes an OTP by key
func (r *OTPRepo) DeleteOTP(ctx context.Context, key string) error {
	return r.client.Del(ctx, key)
}
