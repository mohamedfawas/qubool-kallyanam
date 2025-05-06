// File: internal/adapters/redis/token_repository.go
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

// StoreRefreshToken saves a refresh token for a user with expiration
func (r *TokenRepo) StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error {
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)
	return r.client.Set(ctx, key, token, expiry).Err()
}

// ValidateRefreshToken checks if a refresh token exists and is valid for a user
func (r *TokenRepo) ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error) {
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)
	storedToken, err := r.client.Get(ctx, key).Result()

	if err == redis.Nil {
		// Token not found, so it's invalid
		return false, nil
	} else if err != nil {
		// Redis error
		return false, err
	}

	// Compare the tokens
	return storedToken == token, nil
}

// DeleteRefreshToken removes a refresh token
func (r *TokenRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", refreshTokenPrefix, userID)
	return r.client.Del(ctx, key).Err()
}

// BlacklistToken adds a token to blacklist (for logout)
func (r *TokenRepo) BlacklistToken(ctx context.Context, tokenID string, expiry time.Duration) error {
	key := fmt.Sprintf("%s%s", blacklistPrefix, tokenID)
	return r.client.Set(ctx, key, 1, expiry).Err()
}

// IsTokenBlacklisted checks if a token is blacklisted
func (r *TokenRepo) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("%s%s", blacklistPrefix, tokenID)
	exists, err := r.client.Exists(ctx, key).Result()

	if err != nil {
		return false, err
	}

	return exists > 0, nil
}
