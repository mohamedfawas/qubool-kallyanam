package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	refreshTokenPrefix = "refresh_token:"
	blacklistPrefix    = "blacklist:"
)

// TokenRepo implements the token repository interface using Redis
type TokenRepo struct {
	client *redis.Client
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(client *redis.Client) repositories.TokenRepository {
	return &TokenRepo{
		client: client,
	}
}

// StoreRefreshToken stores a refresh token in Redis with an expiration time.
// Example:
//
//	userID = "123"
//	token = "abc-refresh-token"
//	expiry = 24 hours
//	Redis key will be "refresh_token:123" with value "abc-refresh-token"
func (r *TokenRepo) StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error {
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)

	return r.client.Set(ctx, key, token, expiry).Err()
}

// ValidateRefreshToken checks if the provided token matches the one stored in Redis for a specific user.
// Returns true if valid, false if token doesn't match or doesn't exist.
func (r *TokenRepo) ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error) {
	// Build the key for the refresh token using the user ID
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)

	// Fetch the stored token from Redis
	storedToken, err := r.client.Get(ctx, key).Result()

	// If Redis returns redis.Nil, the key doesn't exist — no token stored
	if err == redis.Nil {
		// Token not found — this usually means it's invalid or expired
		return false, nil
	} else if err != nil {
		// Some Redis error occurred (e.g., connection issue)
		return false, err
	}

	// Compare the tokens
	return storedToken == token, nil
}

// DeleteRefreshToken deletes the stored refresh token from Redis for the given user ID.
// Example: Removes "refresh_token:123" key from Redis
func (r *TokenRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	// Build the key
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)

	// Delete the key from Redis
	return r.client.Del(ctx, key).Err()
}

// BlacklistToken stores a token ID in Redis to mark it as blacklisted (i.e., no longer valid).
// Used when a user logs out or if a token should be forcefully invalidated.
// Example:
//
//	tokenID = "abc-token-id"
//	expiry = 1 hour
//	Redis key = "blacklist:abc-token-id" with value = 1 (dummy value)
func (r *TokenRepo) BlacklistToken(ctx context.Context, tokenID string, expiry time.Duration) error {
	// Build blacklist key
	key := fmt.Sprintf("%s%s", blacklistPrefix, tokenID)

	// Store a dummy value (1) just to indicate presence, with an expiration
	return r.client.Set(ctx, key, 1, expiry).Err()
}

// IsTokenBlacklisted checks if a given token ID is in the blacklist.
// Returns true if blacklisted, false otherwise.
// Example:
//
//	tokenID = "abc-token-id"
//	If "blacklist:abc-token-id" exists in Redis, it means token is blacklisted.
func (r *TokenRepo) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	// Build blacklist key
	key := fmt.Sprintf("%s%s", blacklistPrefix, tokenID)

	// Check if the key exists in Redis
	exists, err := r.client.Exists(ctx, key).Result()

	if err != nil {
		return false, err
	}

	// Redis "Exists" returns 1 if key exists, 0 if not
	return exists > 0, nil
}
