// File: internal/domain/repositories/token_repository.go
package repositories

import (
	"context"
	"time"
)

// TokenRepository defines methods for working with authentication tokens
type TokenRepository interface {
	// StoreRefreshToken saves a refresh token for a user with expiration
	StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error

	// ValidateRefreshToken checks if a refresh token exists and is valid for a user
	ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error)

	// DeleteRefreshToken removes a refresh token
	DeleteRefreshToken(ctx context.Context, userID string) error

	// BlacklistToken adds a token to blacklist (for logout)
	BlacklistToken(ctx context.Context, tokenID string, expiry time.Duration) error

	// IsTokenBlacklisted checks if a token is blacklisted
	IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)
}
