package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

var (
	ErrInvalidInput    = errors.New("invalid input parameters")
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileExists   = errors.New("profile already exists")
	ErrValidation      = errors.New("validation error")
)

type ProfileService struct {
	profileRepo repositories.ProfileRepository
	logger      logging.Logger
}

func NewProfileService(
	profileRepo repositories.ProfileRepository,
	logger logging.Logger,
) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
		logger:      logger,
	}
}

func (s *ProfileService) HandleUserLogin(ctx context.Context, userID string, phone string, lastLogin time.Time) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}

	if exists {
		if err := s.profileRepo.UpdateLastLogin(ctx, userUUID, lastLogin); err != nil {
			return fmt.Errorf("failed to update last login: %w", err)
		}
		s.logger.Info("Updated last login for existing profile", "userID", userID)
		return nil
	}

	now := indianstandardtime.Now()

	// Create minimal profile with only required fields
	// Don't set any of the enum fields - GORM will treat them as NULL
	profile := &models.UserProfile{
		UserID:                userUUID,
		Phone:                 phone,
		IsBride:               false,
		LastLogin:             lastLogin,
		CreatedAt:             now,
		UpdatedAt:             now,
		Community:             models.DefaultNotMentioned,
		MaritalStatus:         models.DefaultNotMentioned,
		Profession:            models.DefaultNotMentioned,
		ProfessionType:        models.DefaultNotMentioned,
		HighestEducationLevel: models.DefaultNotMentioned,
		HomeDistrict:          models.DefaultNotMentioned,
	}

	if err := s.profileRepo.CreateProfile(ctx, profile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	s.logger.Info("Created new profile for first-time login", "userID", userID)
	return nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, isBride bool, fullName string,
	dateOfBirth *time.Time, heightCM *int, physicallyChallenged bool,
	community string, maritalStatus string, profession string,
	professionType string, educationLevel string, homeDistrict string) error {

	// Validate the userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Get existing profile
	existingProfile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if existingProfile == nil {
		return ErrProfileNotFound
	}

	// Validate all fields
	if err := s.validateProfileFields(heightCM, community, maritalStatus, profession,
		professionType, educationLevel, homeDistrict, dateOfBirth); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// Update the profile
	existingProfile.IsBride = isBride
	existingProfile.FullName = fullName
	existingProfile.DateOfBirth = dateOfBirth
	existingProfile.HeightCM = heightCM
	existingProfile.PhysicallyChallenged = physicallyChallenged

	// Convert string fields to their respective types
	existingProfile.Community = models.Community(community)
	existingProfile.MaritalStatus = models.MaritalStatus(maritalStatus)
	existingProfile.Profession = models.Profession(profession)
	existingProfile.ProfessionType = models.ProfessionType(professionType)
	existingProfile.HighestEducationLevel = models.EducationLevel(educationLevel)
	existingProfile.HomeDistrict = models.HomeDistrict(homeDistrict)

	// Update the updated_at timestamp
	existingProfile.UpdatedAt = indianstandardtime.Now()

	// Save the updated profile
	if err := s.profileRepo.UpdateProfile(ctx, existingProfile); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	s.logger.Info("Profile updated successfully", "userID", userID)
	return nil
}

// Add validation helper function
func (s *ProfileService) validateProfileFields(heightCM *int, community string, maritalStatus string,
	profession string, professionType string, educationLevel string,
	homeDistrict string, dateOfBirth *time.Time) error {

	if err := validation.ValidateHeight(heightCM); err != nil {
		return err
	}

	if err := validation.ValidateCommunity(community); err != nil {
		return err
	}

	if err := validation.ValidateMaritalStatus(maritalStatus); err != nil {
		return err
	}

	if err := validation.ValidateProfession(profession); err != nil {
		return err
	}

	if err := validation.ValidateProfessionType(professionType); err != nil {
		return err
	}

	if err := validation.ValidateEducationLevel(educationLevel); err != nil {
		return err
	}

	if err := validation.ValidateHomeDistrict(homeDistrict); err != nil {
		return err
	}

	if err := validation.ValidateDateOfBirth(dateOfBirth); err != nil {
		return err
	}

	return nil
}
