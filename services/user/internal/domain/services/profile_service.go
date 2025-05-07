// user/internal/domain/services/profile_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

var (
	ErrInvalidInput    = errors.New("invalid input parameters")
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileExists   = errors.New("profile already exists")
)

// ProfileService handles user profile operations
type ProfileService struct {
	profileRepo repositories.ProfileRepository
	logger      logging.Logger
}

// NewProfileService creates a new profile service
func NewProfileService(
	profileRepo repositories.ProfileRepository,
	logger logging.Logger,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		logger:      logger,
	}
}

// HandleUserLogin processes user login events
// Creates a profile if it doesn't exist or updates last_login if it does
func (s *ProfileService) HandleUserLogin(ctx context.Context, userID string, phone string, lastLogin time.Time) error {
	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}

	// If profile exists, just update last login time
	if exists {
		if err := s.profileRepo.UpdateLastLogin(ctx, userUUID, lastLogin); err != nil {
			return fmt.Errorf("failed to update last login: %w", err)
		}
		s.logger.Info("Updated last login for existing profile", "userID", userID)
		return nil
	}

	// Create minimal profile for first-time login
	now := indianstandardtime.Now()
	profile := &models.UserProfile{
		UserID:    userUUID,
		Phone:     phone,
		LastLogin: lastLogin,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.profileRepo.CreateProfile(ctx, profile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	s.logger.Info("Created new profile for first-time login", "userID", userID)
	return nil
}
