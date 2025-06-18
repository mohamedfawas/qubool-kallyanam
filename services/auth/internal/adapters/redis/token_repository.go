package redis

import (
	"context"
	"fmt"
	"time"

	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	"github.com/redis/go-redis/v9"
)

type TokenRepo struct {
	client *redisdb.Client
}

func NewTokenRepository(client *redisdb.Client) repositories.TokenRepository {
	return &TokenRepo{
		client: client,
	}
}

// StoreRefreshToken stores a refresh token in Redis with an expiration time.
func (r *TokenRepo) StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error {
	key := fmt.Sprintf("%s%s", constants.RefreshTokenPrefix, userID)

	return r.client.Set(ctx, key, token, expiry)
}

// ValidateRefreshToken checks if the provided token matches the one stored in Redis for a specific user.
func (r *TokenRepo) ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error) {
	key := fmt.Sprintf("%s%s", constants.RefreshTokenPrefix, userID)

	storedToken, err := r.client.Get(ctx, key)
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return storedToken == token, nil
}

// DeleteRefreshToken deletes the stored refresh token from Redis for the given user ID.
func (r *TokenRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", constants.RefreshTokenPrefix, userID)
	return r.client.Del(ctx, key)
}

// BlacklistToken stores a token ID in Redis to mark it as blacklisted.
func (r *TokenRepo) BlacklistToken(ctx context.Context, tokenID string, expiry time.Duration) error {
	key := fmt.Sprintf("%s%s", constants.BlacklistPrefix, tokenID)
	return r.client.Set(ctx, key, "1", expiry)
}

// IsTokenBlacklisted checks if a given token ID is in the blacklist.
func (r *TokenRepo) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("%s%s", constants.BlacklistPrefix, tokenID)

	// Check if key exists using Get method
	_, err := r.client.Get(ctx, key)
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}
