package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	autherrors "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/errors"
)

// TokenPair represents JWT access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32 // Expiration time in seconds
}

type AuthService struct {
	userRepo        repositories.UserRepository
	tokenRepo       repositories.TokenRepository
	adminRepo       repositories.AdminRepository
	jwtManager      *jwt.Manager
	logger          logging.Logger
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	messageBroker   *rabbitmq.Client
}

func NewAuthService(
	userRepo repositories.UserRepository,
	tokenRepo repositories.TokenRepository,
	adminRepo repositories.AdminRepository,
	jwtManager *jwt.Manager,
	logger logging.Logger,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	messageBroker *rabbitmq.Client,
) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		adminRepo:       adminRepo,
		jwtManager:      jwtManager,
		logger:          logger,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		messageBroker:   messageBroker,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetUser(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to retrieve user", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	if user == nil {
		s.logger.Debug("User not found", "email", email)
		return nil, autherrors.ErrInvalidCredentials
	}

	if !user.IsActive {
		s.logger.Debug("Account is disabled", "email", email)
		return nil, autherrors.ErrAccountDisabled
	}

	if !encryption.VerifyPassword(user.PasswordHash, password) {
		s.logger.Debug("Invalid password", "email", email)
		return nil, autherrors.ErrInvalidCredentials
	}

	if !user.Verified {
		s.logger.Debug("Account not verified", "email", email)
		return nil, autherrors.ErrAccountNotVerified
	}

	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "userId", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	err = s.userRepo.UpdateLastLogin(ctx, user.ID.String())
	if err != nil {
		s.logger.Error("Failed to update last login time", "userId", user.ID, "error", err)
	}

	// Send login event
	if s.messageBroker != nil {
		loginEvent := map[string]interface{}{
			"user_id":    user.ID.String(),
			"phone":      user.Phone,
			"email":      user.Email,
			"last_login": indianstandardtime.Now(),
			"event_type": constants.EventTypeLogin,
		}

		if err := s.messageBroker.Publish(constants.TopicUserLogin, loginEvent); err != nil {
			s.logger.Error("Failed to publish login event", "userId", user.ID, "error", err)
		} else {
			s.logger.Info("Login event published", "userId", user.ID)
		}
	}

	s.logger.Info("User logged in successfully", "email", email, "userId", user.ID)
	return tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		s.logger.Debug("Logout failed - invalid token")
		return autherrors.ErrInvalidToken
	}

	tokenID := claims.ID
	isBlacklisted, err := s.tokenRepo.IsTokenBlacklisted(ctx, tokenID)
	if err != nil {
		s.logger.Error("Failed to check token blacklist status", "error", err)
		return fmt.Errorf("failed to check token blacklist status: %w", err)
	}

	if isBlacklisted {
		s.logger.Debug("Logout failed - token already blacklisted")
		return autherrors.ErrInvalidToken
	}

	expiryTime := time.Until(time.Unix(claims.ExpiresAt.Time.Unix(), 0))
	err = s.tokenRepo.BlacklistToken(ctx, tokenID, expiryTime)
	if err != nil {
		s.logger.Error("Failed to blacklist token", "error", err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	userID := claims.UserID
	err = s.tokenRepo.DeleteRefreshToken(ctx, fmt.Sprintf("%d", userID))
	if err != nil {
		s.logger.Error("Failed to delete refresh token", "error", err, "userId", userID)
	}

	s.logger.Info("User logged out successfully", "userId", userID)
	return nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		s.logger.Debug("Refresh token validation failed", "error", err)
		return nil, autherrors.ErrInvalidRefreshToken
	}

	userID := claims.UserID
	if userID == "" {
		s.logger.Debug("Refresh token missing user ID claim")
		return nil, autherrors.ErrInvalidRefreshToken
	}

	valid, err := s.tokenRepo.ValidateRefreshToken(ctx, userID, refreshToken)
	if err != nil {
		s.logger.Error("Error validating refresh token", "error", err)
		return nil, fmt.Errorf("error validating refresh token: %w", err)
	}
	if !valid {
		s.logger.Debug("Refresh token does not match stored token")
		return nil, autherrors.ErrInvalidRefreshToken
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Invalid UUID format in token", "error", err)
		return nil, autherrors.ErrInvalidRefreshToken
	}

	newTokens, err := s.generateTokens(userUUID)
	if err != nil {
		s.logger.Error("Failed to generate new tokens", "error", err)
		return nil, err
	}

	if err := s.tokenRepo.DeleteRefreshToken(ctx, userID); err != nil {
		s.logger.Error("Failed to delete old refresh token", "error", err)
	}

	s.logger.Info("Tokens refreshed successfully", "userID", userID)
	return newTokens, nil
}

func (s *AuthService) generateTokens(userID uuid.UUID) (*TokenPair, error) {
	userIDStr := userID.String()
	user, err := s.userRepo.GetUser(context.Background(), "id", userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get user for token generation: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found for token generation")
	}

	role := jwt.RoleUser
	if user.IsPremium() {
		role = jwt.RolePremiumUser
	}

	// Create access token (short-lived)
	accessToken, err := s.jwtManager.GenerateAccessToken(
		userIDStr,
		role, //Assigning role "user" or "premium user"
		true, // Indicates it’s a refreshable token
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Create refresh token (long-lived)
	refreshToken, err := s.jwtManager.GenerateRefreshToken(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in DB for future validation
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

func (s *AuthService) Delete(ctx context.Context, userID string, password string) error {
	user, err := s.userRepo.GetUser(ctx, "id", userID)
	if err != nil {
		s.logger.Error("Failed to retrieve user", "userID", userID, "error", err)
		return fmt.Errorf("error retrieving user: %w", err)
	}
	if user == nil {
		s.logger.Debug("User not found", "userID", userID)
		return autherrors.ErrUserNotFound
	}

	if !encryption.VerifyPassword(user.PasswordHash, password) {
		s.logger.Debug("Invalid password for account deletion", "userID", userID)
		return autherrors.ErrInvalidCredentials
	}

	err = s.userRepo.SoftDeleteUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to soft delete user", "userID", userID, "error", err)
		return fmt.Errorf("failed to delete user account: %w", err)
	}

	if err := s.tokenRepo.DeleteRefreshToken(ctx, userID); err != nil {
		s.logger.Error("Failed to delete refresh token", "userID", userID, "error", err)
	}

	// Publish an account deletion event to the message broker
	if s.messageBroker != nil {
		deleteEvent := map[string]interface{}{
			"user_id":    userID,
			"event_type": constants.EventTypeUserDeleted,
			"timestamp":  indianstandardtime.Now(),
		}
		if err := s.messageBroker.Publish(constants.TopicUserDeleted, deleteEvent); err != nil {
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
	err := s.userRepo.UpdatePremiumUntil(ctx, userID, premiumUntil)
	if err != nil {
		s.logger.Error("Failed to update premium status", "userID", userID, "error", err)
		return fmt.Errorf("failed to update premium status: %w", err)
	}
	s.logger.Info("Premium status updated successfully", "userID", userID)
	return nil
}

func (s *AuthService) AdminLogin(ctx context.Context, email, password string) (*TokenPair, error) {
	admin, err := s.adminRepo.GetAdminByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to retrieve admin", "email", email, "error", err)
		return nil, fmt.Errorf("error retrieving admin: %w", err)
	}
	if admin == nil {
		s.logger.Debug("Admin not found", "email", email)
		return nil, autherrors.ErrAdminNotFound
	}

	if !admin.IsActive {
		s.logger.Debug("Admin account is disabled", "email", email)
		return nil, autherrors.ErrAdminAccountDisabled
	}

	if !encryption.VerifyPassword(admin.PasswordHash, password) {
		s.logger.Debug("Invalid admin password", "email", email)
		return nil, autherrors.ErrInvalidCredentials
	}

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

// GetUsersList retrieves users list for admin (reuse existing patterns)
func (s *AuthService) GetUsersList(ctx context.Context, params repositories.GetUsersParams) ([]*models.User, int64, error) {
	s.logger.Info("Getting users list", "limit", params.Limit, "offset", params.Offset)

	users, total, err := s.userRepo.GetUsers(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get users from repository", "error", err)
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	s.logger.Info("Successfully retrieved users", "count", len(users), "total", total)
	return users, total, nil
}

// GetUserByIdentifier retrieves single user by field (reuse existing GetUser pattern)
func (s *AuthService) GetUserByIdentifier(ctx context.Context, field, value string) (*models.User, error) {
	s.logger.Info("Getting user by identifier", "field", field, "value", value)

	// Reuse existing repository method
	user, err := s.userRepo.GetUser(ctx, field, value)
	if err != nil {
		s.logger.Error("Failed to get user from repository", "error", err, "field", field)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	s.logger.Info("Successfully retrieved user", "found", user != nil)
	return user, nil
}
