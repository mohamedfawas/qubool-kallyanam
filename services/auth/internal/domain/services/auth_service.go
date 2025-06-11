package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

// Define authentication-related errors
var (
	ErrUserNotFound         = errors.New("user not found")
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
	messageBroker    *rabbitmq.Client
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
	messageBroker *rabbitmq.Client,
) *AuthService {
	return &AuthService{
		registrationRepo: registrationRepo,
		tokenRepo:        tokenRepo,
		adminRepo:        adminRepo,
		jwtManager:       jwtManager,
		logger:           logger,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		messageBroker:    messageBroker,
	}
}

// Login authenticates a user with email and password
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	// 1. Try to retrieve the user by email from the database (from "users" table)
	user, err := s.registrationRepo.GetUser(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to retrieve user", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	// 2. If user is not found, return error
	if user == nil {
		s.logger.Debug("User not found", "email", email)
		return nil, ErrInvalidCredentials
	}

	// 3. Check if user's account is active
	if !user.IsActive {
		s.logger.Debug("Account is disabled", "email", email)
		return nil, ErrAccountDisabled
	}

	// 4. Check if the given password matches the hashed password in DB
	if !encryption.VerifyPassword(user.PasswordHash, password) {
		s.logger.Debug("Invalid password", "email", email)
		return nil, ErrInvalidCredentials
	}

	// 5. Check if user has verified their email
	if !user.Verified {
		s.logger.Debug("Account not verified", "email", email)
		return nil, ErrAccountNotVerified
	}

	// 6. If everything is okay, generate access and refresh tokens
	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "userId", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// 7. Update the last login timestamp for the user
	err = s.registrationRepo.UpdateLastLogin(ctx, user.ID.String())
	if err != nil {
		s.logger.Error("Failed to update last login time", "userId", user.ID, "error", err)
		// Continue despite this error since authentication succeeded
	}

	// 8. Send login event to message broker
	// creates respective entry of the "user_id" in "user_profile" table of "user" service
	// also used to update "last_login" field in "user_profile" table of "user" service)
	if s.messageBroker != nil {
		loginEvent := map[string]interface{}{
			"user_id":    user.ID.String(),
			"phone":      user.Phone,
			"email":      user.Email,
			"last_login": indianstandardtime.Now(),
			"event_type": "login", // Can help identify this is a login event
		}

		if err := s.messageBroker.Publish("user.login", loginEvent); err != nil {
			s.logger.Error("Failed to publish login event", "userId", user.ID, "error", err)
			// Continue despite this error since authentication succeeded
		} else {
			s.logger.Info("Login event published", "userId", user.ID)
		}
	}

	// 9. Return the tokens to the client
	s.logger.Info("User logged in successfully", "email", email, "userId", user.ID)
	return tokens, nil
}

// Logout will invalidate the user's token (used for secure logout)
func (s *AuthService) Logout(ctx context.Context, accessToken string) error {
	// 1. Validate the token to extract claims
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		s.logger.Debug("Logout failed - invalid token")
		return ErrInvalidToken
	}

	// 2. Get the token ID from claims (used to blacklist it)
	tokenID := claims.ID

	// 3. Check if this token is already blacklisted (possibly reused)
	isBlacklisted, err := s.tokenRepo.IsTokenBlacklisted(ctx, tokenID)
	if err != nil {
		s.logger.Error("Failed to check token blacklist status", "error", err)
		return fmt.Errorf("failed to check token blacklist status: %w", err)
	}

	if isBlacklisted {
		s.logger.Debug("Logout failed - token already blacklisted")
		return ErrInvalidToken
	}

	// 4. Calculate how long the token is valid for
	expiryTime := time.Until(time.Unix(claims.ExpiresAt.Time.Unix(), 0))

	// 5. Blacklist this token for the remaining duration
	err = s.tokenRepo.BlacklistToken(ctx, tokenID, expiryTime)
	if err != nil {
		s.logger.Error("Failed to blacklist token", "error", err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	// 6. Also delete the refresh token to prevent reuse
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
	// 1. Validate the refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		s.logger.Debug("Refresh token validation failed", "error", err)
		return nil, ErrInvalidRefreshToken
	}

	// 2. Ensure the token has user ID in claims
	userID := claims.UserID
	if userID == "" {
		s.logger.Debug("Refresh token missing user ID claim")
		return nil, ErrInvalidRefreshToken
	}

	// 3. Check if refresh token exists in redis
	valid, err := s.tokenRepo.ValidateRefreshToken(ctx, userID, refreshToken)
	if err != nil {
		s.logger.Error("Error validating refresh token", "error", err)
		return nil, fmt.Errorf("error validating refresh token: %w", err)
	}

	if !valid {
		s.logger.Debug("Refresh token does not match stored token")
		return nil, ErrInvalidRefreshToken
	}

	// 4. Convert string userID to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Invalid UUID format in token", "error", err)
		return nil, ErrInvalidRefreshToken
	}

	// 5. Generate new tokens
	newTokens, err := s.generateTokens(userUUID)
	if err != nil {
		s.logger.Error("Failed to generate new tokens", "error", err)
		return nil, err
	}

	// 6. Delete old refresh token to avoid reuse
	if err := s.tokenRepo.DeleteRefreshToken(ctx, userID); err != nil {
		s.logger.Error("Failed to delete old refresh token", "error", err)
		// Continue despite this error
	}

	s.logger.Info("Tokens refreshed successfully", "userID", userID)
	return newTokens, nil
}

// generateTokens creates new access and refresh tokens for a user
func (s *AuthService) generateTokens(userID uuid.UUID) (*TokenPair, error) {
	userIDStr := userID.String()

	// Get user from database to check premium status
	user, err := s.registrationRepo.GetUser(context.Background(), "id", userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get user for token generation: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found for token generation")
	}

	// Determine role based on premium status
	role := jwt.RoleUser
	if user.IsPremium() {
		role = jwt.RolePremiumUser
	}

	// 1. Create access token (short-lived)
	accessToken, err := s.jwtManager.GenerateAccessToken(
		userIDStr,
		role, //Assigning role "user"
		true, // Indicates it’s a refreshable token
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 2. Create refresh token (long-lived)
	refreshToken, err := s.jwtManager.GenerateRefreshToken(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 3. Store refresh token in DB for future validation
	err = s.tokenRepo.StoreRefreshToken(
		context.Background(),
		userIDStr,
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
	// 1. Retrieve admin details by email
	admin, err := s.adminRepo.GetAdminByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to retrieve admin", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving admin: %w", err)
	}

	// 2. Check if admin exists
	if admin == nil {
		s.logger.Debug("Admin not found", "email", email)
		return nil, ErrAdminNotFound
	}

	// 3. Ensure admin account is active
	if !admin.IsActive {
		s.logger.Debug("Admin account is disabled", "email", email)
		return nil, ErrAdminAccountDisabled
	}

	// 4. Check password correctness
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

func (s *AuthService) generateAdminTokens(adminID uuid.UUID) (*TokenPair, error) {
	adminIDStr := adminID.String()

	accessToken, err := s.jwtManager.GenerateAccessToken(
		adminIDStr,
		jwt.RoleAdmin,
		true,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate admin access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(adminIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate admin refresh token: %w", err)
	}

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

func (s *AuthService) Delete(ctx context.Context, userID string, password string) error {
	// Retrieve the user from the registration repository
	user, err := s.registrationRepo.GetUser(ctx, "id", userID)
	if err != nil {
		s.logger.Error("Failed to retrieve user", "userID", userID, "error", err)
		return fmt.Errorf("error retrieving user: %w", err)
	}

	// If user not found, return not found error
	if user == nil {
		s.logger.Debug("User not found", "userID", userID)
		return ErrUserNotFound
	}

	// Verify that the password provided matches the stored hash
	if !encryption.VerifyPassword(user.PasswordHash, password) {
		s.logger.Debug("Invalid password for account deletion", "userID", userID)
		return ErrInvalidCredentials
	}

	// Soft delete the user
	err = s.registrationRepo.SoftDeleteUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to soft delete user", "userID", userID, "error", err)
		return fmt.Errorf("failed to delete user account: %w", err)
	}

	// Attempt to delete the refresh token associated with the user
	if err := s.tokenRepo.DeleteRefreshToken(ctx, userID); err != nil {
		s.logger.Error("Failed to delete refresh token", "userID", userID, "error", err)
	}

	// Publish an account deletion event to the message broker
	if s.messageBroker != nil {
		deleteEvent := map[string]interface{}{
			"user_id":    userID,
			"event_type": "user.deleted",
			"timestamp":  indianstandardtime.Now(),
		}
		if err := s.messageBroker.Publish("user.deleted", deleteEvent); err != nil {
			s.logger.Error("Failed to publish account deletion event", "userID", userID, "error", err)
		} else {
			s.logger.Info("Account deletion event published", "userID", userID)
		}
	}

	s.logger.Info("User account soft deleted successfully", "userID", userID)
	return nil
}

func (s *AuthService) UpdateUserPremiumStatus(ctx context.Context, userID string, premiumUntil time.Time) error {
	s.logger.Info("Updating user premium status", "userID", userID, "premiumUntil", premiumUntil)

	err := s.registrationRepo.UpdatePremiumUntil(ctx, userID, premiumUntil)
	if err != nil {
		s.logger.Error("Failed to update premium status", "userID", userID, "error", err)
		return fmt.Errorf("failed to update premium status: %w", err)
	}

	s.logger.Info("Premium status updated successfully", "userID", userID)
	return nil
}
