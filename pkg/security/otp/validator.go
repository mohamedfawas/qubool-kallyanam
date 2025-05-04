package otp

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidOTP       = errors.New("invalid or expired OTP")
	ErrGenerationFailed = errors.New("failed to generate OTP")
)

// Store handles OTP storage and validation
type Store struct {
	redisClient *redis.Client
	config      Config
}

// NewStore creates a new OTP store
func NewStore(redisClient *redis.Client, config Config) *Store {
	return &Store{
		redisClient: redisClient,
		config:      config,
	}
}

// StoreOTP saves an OTP to Redis with expiry
func (s *Store) StoreOTP(ctx context.Context, identifier, otp string) error {
	key := "otp:" + identifier
	return s.redisClient.Set(ctx, key, otp, s.config.ExpiryTime).Err()
}

// ValidateOTP checks if the provided OTP is valid
// If valid, it deletes the OTP to prevent reuse
func (s *Store) ValidateOTP(ctx context.Context, identifier, otp string) (bool, error) {
	key := "otp:" + identifier

	// Get the stored OTP
	storedOTP, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, ErrInvalidOTP
		}
		return false, err
	}

	// Compare the OTPs
	if storedOTP != otp {
		return false, nil
	}

	// Delete the OTP to prevent reuse
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		return true, err // OTP was valid but failed to delete
	}

	return true, nil
}
