// File: internal/domain/services/auth_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

// Define authentication-related errors
var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrAccountNotVerified   = errors.New("account is not verified")
	ErrAccountDisabled      = errors.New("account is disabled")
	ErrInvalidToken         = errors.New("invalid or expired token")
	ErrInvalidRefreshToken  = errors.New("invalid or expired refresh token")
	ErrAdminNotFound        = errors.New("admin not found")
	ErrAdminAccountDisabled = errors.New("admin account is disabled")
)

// TokenPair represents JWT access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32 // Expiration time in seconds
}

// AuthService handles authentication operations
type AuthService struct {
	registrationRepo repositories.RegistrationRepository
	tokenRepo        repositories.TokenRepository
	adminRepo        repositories.AdminRepository
	jwtManager       *jwt.Manager
	logger           logging.Logger
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(
	registrationRepo repositories.RegistrationRepository,
	tokenRepo repositories.TokenRepository,
	adminRepo repositories.AdminRepository,
	jwtManager *jwt.Manager,
	logger logging.Logger,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		registrationRepo: registrationRepo,
		tokenRepo:        tokenRepo,
		adminRepo:        adminRepo,
		jwtManager:       jwtManager,
		logger:           logger,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// Login authenticates a user with email and password
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	// Get user by email
	user, err := s.registrationRepo.GetUser(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to retrieve user", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	// Check if user exists
	if user == nil {
		s.logger.Debug("User not found", "email", email)
		return nil, ErrInvalidCredentials
	}

	// Check if account is active
	if !user.IsActive {
		s.logger.Debug("Account is disabled", "email", email)
		return nil, ErrAccountDisabled
	}

	// Verify password
	if !encryption.VerifyPassword(user.PasswordHash, password) {
		s.logger.Debug("Invalid password", "email", email)
		return nil, ErrInvalidCredentials
	}

	// Check if account is verified
	if !user.Verified {
		s.logger.Debug("Account not verified", "email", email)
		return nil, ErrAccountNotVerified
	}

	// Generate tokens
	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "userId", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login time
	err = s.registrationRepo.UpdateLastLogin(ctx, user.ID.String())
	if err != nil {
		s.logger.Error("Failed to update last login time", "userId", user.ID, "error", err)
		// Continue despite this error since authentication succeeded
	}

	s.logger.Info("User logged in successfully", "email", email, "userId", user.ID)
	return tokens, nil
}

// Logout invalidates a user's tokens
func (s *AuthService) Logout(ctx context.Context, accessToken string) error {
	// Validate the access token
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		s.logger.Debug("Logout failed - invalid token")
		return ErrInvalidToken
	}

	// Get expiration time from token
	expiryTime := time.Until(time.Unix(claims.ExpiresAt.Time.Unix(), 0))

	// Get token ID from JWT claims (jti claim)
	tokenID := claims.ID

	// Add token to blacklist
	err = s.tokenRepo.BlacklistToken(ctx, tokenID, expiryTime)
	if err != nil {
		s.logger.Error("Failed to blacklist token", "error", err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// Delete refresh token for user
	userID := claims.UserID
	err = s.tokenRepo.DeleteRefreshToken(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		s.logger.Error("Failed to delete refresh token", "error", err, "userId", userID)
		// Continue despite this error since blacklisting was successful
	}

	s.logger.Info("User logged out successfully", "userId", userID)
	return nil
}

// RefreshToken validates a refresh token and issues new tokens
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate the refresh token format and signature
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		s.logger.Debug("Refresh token validation failed", "error", err)
		return nil, ErrInvalidRefreshToken
	}

	// Extract user ID string from token claims
	userIDStr := claims.UserIDString
	if userIDStr == "" {
		s.logger.Debug("Refresh token missing user ID string claim")
		return nil, ErrInvalidRefreshToken
	}

	// Check if the refresh token is valid in Redis
	valid, err := s.tokenRepo.ValidateRefreshToken(ctx, userIDStr, refreshToken)
	if err != nil {
		s.logger.Error("Error validating refresh token", "error", err)
		return nil, fmt.Errorf("error validating refresh token: %w", err)
	}

	if !valid {
		s.logger.Debug("Refresh token does not match stored token")
		return nil, ErrInvalidRefreshToken
	}

	// Parse the UUID from the string
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		s.logger.Error("Invalid UUID format in token", "error", err)
		return nil, ErrInvalidRefreshToken
	}

	// Generate new tokens using the UUID
	newTokens, err := s.generateTokens(userUUID)
	if err != nil {
		s.logger.Error("Failed to generate new tokens", "error", err)
		return nil, err
	}

	// Delete the old refresh token
	if err := s.tokenRepo.DeleteRefreshToken(ctx, userIDStr); err != nil {
		s.logger.Error("Failed to delete old refresh token", "error", err)
		// Continue despite this error
	}

	s.logger.Info("Tokens refreshed successfully", "userID", userIDStr)
	return newTokens, nil
}

// generateTokens creates a pair of JWT tokens (access and refresh) for the user
func (s *AuthService) generateTokens(userID uuid.UUID) (*TokenPair, error) {
	// Store the UUID string in the custom claim
	userIDStr := userID.String()

	// For JWT standard claims we'll use a simple numeric ID (0 is fine for MVP)
	userIDUint := uint(0)

	// Determine user role (basic implementation for MVP)
	role := jwt.RoleUser
	var premiumUntil *int64 = nil // Can be enhanced for premium users

	// Generate access token with the UUID string in a custom claim
	accessToken, err := s.jwtManager.GenerateAccessToken(
		userIDUint,
		role,
		true,
		premiumUntil,
		userIDStr, // Pass the UUID string to store in the token
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token with the same UUID string in a custom claim
	refreshToken, err := s.jwtManager.GenerateRefreshToken(
		userIDUint,
		userIDStr, // Pass the UUID string to store in the token
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Redis using the UUID string as the key
	// This ensures we can look it up correctly later
	err = s.tokenRepo.StoreRefreshToken(
		context.Background(),
		userIDStr, // Use the actual UUID string as the key
		refreshToken,
		s.refreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int32(s.accessTokenTTL.Seconds()),
	}, nil
}

// AdminLogin authenticates an admin with email and password
func (s *AuthService) AdminLogin(ctx context.Context, email, password string) (*TokenPair, error) {
	// Get admin by email
	admin, err := s.adminRepo.GetAdminByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to retrieve admin", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving admin: %w", err)
	}

	// Check if admin exists
	if admin == nil {
		s.logger.Debug("Admin not found", "email", email)
		return nil, ErrAdminNotFound
	}

	// Check if admin account is active
	if !admin.IsActive {
		s.logger.Debug("Admin account is disabled", "email", email)
		return nil, ErrAdminAccountDisabled
	}

	// Verify password
	if !encryption.VerifyPassword(admin.PasswordHash, password) {
		s.logger.Debug("Invalid admin password", "email", email)
		return nil, ErrInvalidCredentials
	}

	// Generate tokens with ADMIN role
	tokens, err := s.generateAdminTokens(admin.ID)
	if err != nil {
		s.logger.Error("Failed to generate admin tokens", "adminId", admin.ID, "error", err)
		return nil, fmt.Errorf("failed to generate admin tokens: %w", err)
	}

	s.logger.Info("Admin logged in successfully", "email", email, "adminId", admin.ID)
	return tokens, nil
}

// Generate tokens specifically for admins
func (s *AuthService) generateAdminTokens(adminID uuid.UUID) (*TokenPair, error) {
	// Store the UUID string in the custom claim
	adminIDStr := adminID.String()

	// For JWT standard claims we'll use a simple numeric ID
	userIDUint := uint(0)

	// Use ADMIN role for admin users
	role := jwt.RoleAdmin

	// Generate access token with the UUID string in a custom claim
	accessToken, err := s.jwtManager.GenerateAccessToken(
		userIDUint,
		role,
		true,
		nil,        // No premium until for admins
		adminIDStr, // Pass the UUID string to store in the token
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate admin access token: %w", err)
	}

	// Generate refresh token with the same UUID string in a custom claim
	refreshToken, err := s.jwtManager.GenerateRefreshToken(
		userIDUint,
		adminIDStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate admin refresh token: %w", err)
	}

	// Store refresh token in Redis using the UUID string as the key
	err = s.tokenRepo.StoreRefreshToken(
		context.Background(),
		adminIDStr,
		refreshToken,
		s.refreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to store admin refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int32(s.accessTokenTTL.Seconds()),
	}, nil
}
