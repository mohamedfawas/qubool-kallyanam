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
	"gorm.io/gorm"
)

var (
	ErrInvalidInput    = errors.New("invalid input parameters")
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileExists   = errors.New("profile already exists")
	ErrValidation      = errors.New("validation error")
)

type DetailedProfileData struct {
	ID                    uint
	IsBride               bool
	FullName              string
	DateOfBirth           *time.Time
	HeightCM              *int
	PhysicallyChallenged  bool
	Community             models.Community
	MaritalStatus         models.MaritalStatus
	Profession            models.Profession
	ProfessionType        models.ProfessionType
	HighestEducationLevel models.EducationLevel
	HomeDistrict          models.HomeDistrict
	ProfilePictureURL     *string
	LastLogin             time.Time
	PartnerPreferences    *models.PartnerPreferences
	AdditionalPhotos      []*models.UserPhoto
	IntroVideo            *models.UserVideo
	Age                   int
}

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

func (s *ProfileService) HandleUserLogin(ctx context.Context, userID string, phone string, email string, lastLogin time.Time) error {
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
		if err := s.profileRepo.UpdateEmail(ctx, userUUID, email); err != nil {
			s.logger.Warn("Failed to update email", "userID", userID, "error", err)
			// Don't fail the login for this
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
		Email:                 email,
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

func (s *ProfileService) HandleUserDeletion(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Invalid user ID format", "userID", userID, "error", err)
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		s.logger.Error("Failed to retrieve user profile for deletion", "userID", userID, "error", err)
		return fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	if profile == nil {
		s.logger.Info("User profile not found for deletion - already deleted or never existed", "userID", userID)
		return nil
	}

	if err := s.profileRepo.SoftDeleteUserProfile(ctx, userUUID); err != nil {
		s.logger.Error("Failed to soft delete user profile", "userID", userID, "error", err)
		return fmt.Errorf("failed to soft delete user profile: %w", err)
	}

	if err := s.profileRepo.SoftDeletePartnerPreferences(ctx, profile.ID); err != nil {
		s.logger.Error("Failed to soft delete partner preferences", "userID", userID, "profileID", profile.ID, "error", err)
		return fmt.Errorf("failed to soft delete partner preferences: %w", err)
	}

	s.logger.Info("Successfully soft deleted user data", "userID", userID, "profileID", profile.ID)
	return nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, isBride bool, fullName string,
	dateOfBirth string, heightCM *int, physicallyChallenged bool,
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

	var dobTime *time.Time
	if dateOfBirth != "" {
		parsedTime, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: invalid date format: %v", ErrInvalidInput, err)
		}
		dobTime = &parsedTime
	}

	// Update the profile
	existingProfile.IsBride = isBride
	existingProfile.FullName = fullName
	existingProfile.DateOfBirth = dobTime
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
	homeDistrict string, dateOfBirth string) error {

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

func (s *ProfileService) PatchProfile(ctx context.Context, userID string,
	isBride *bool, fullName *string, dateOfBirth string, heightCM *int,
	physicallyChallenged *bool, community *string, maritalStatus *string,
	profession *string, professionType *string, educationLevel *string,
	homeDistrict *string, clearDateOfBirth bool, clearHeightCM bool) error {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return ErrProfileNotFound
	}

	// Validate the fields that are being updated
	if heightCM != nil {
		if err := validation.ValidateHeight(heightCM); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if community != nil {
		if err := validation.ValidateCommunity(*community); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if maritalStatus != nil {
		if err := validation.ValidateMaritalStatus(*maritalStatus); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if profession != nil {
		if err := validation.ValidateProfession(*profession); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if professionType != nil {
		if err := validation.ValidateProfessionType(*professionType); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if educationLevel != nil {
		if err := validation.ValidateEducationLevel(*educationLevel); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if homeDistrict != nil {
		if err := validation.ValidateHomeDistrict(*homeDistrict); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	if dateOfBirth != "" {
		if err := validation.ValidateDateOfBirth(dateOfBirth); err != nil {
			return fmt.Errorf("%w: %v", ErrValidation, err)
		}
	}

	// Build updates map
	updates := make(map[string]interface{})

	if isBride != nil {
		updates["is_bride"] = *isBride
	}

	if fullName != nil {
		updates["full_name"] = *fullName
	}

	if dateOfBirth != "" {
		// Convert string date to time.Time for database storage
		parsedTime, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: invalid date format: %v", ErrInvalidInput, err)
		}
		updates["date_of_birth"] = parsedTime
	} else if clearDateOfBirth {
		updates["date_of_birth"] = nil
	}

	if heightCM != nil {
		updates["height_cm"] = *heightCM
	} else if clearHeightCM {
		updates["height_cm"] = nil
	}

	if physicallyChallenged != nil {
		updates["physically_challenged"] = *physicallyChallenged
	}

	if community != nil {
		updates["community"] = *community
	}

	if maritalStatus != nil {
		updates["marital_status"] = *maritalStatus
	}

	if profession != nil {
		updates["profession"] = *profession
	}

	if professionType != nil {
		updates["profession_type"] = *professionType
	}

	if educationLevel != nil {
		updates["highest_education_level"] = *educationLevel
	}

	if homeDistrict != nil {
		updates["home_district"] = *homeDistrict
	}

	// If no updates, return success
	if len(updates) == 0 {
		return nil
	}

	if err := s.profileRepo.PatchProfile(ctx, userUUID, updates); err != nil {
		return fmt.Errorf("failed to patch profile: %w", err)
	}

	s.logger.Info("Profile patched successfully", "userID", userID, "updatedFields", len(updates))
	return nil
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving profile: %w", err)
	}

	if profile == nil {
		return nil, ErrProfileNotFound
	}

	return profile, nil
}

// GetUserUUIDByProfileID resolves public profile ID to user UUID
func (s *ProfileService) GetUserUUIDByProfileID(ctx context.Context, profileID uint64) (string, error) {
	if profileID == 0 {
		return "", fmt.Errorf("%w: profile ID cannot be zero", ErrInvalidInput)
	}

	userUUID, err := s.profileRepo.GetUserUUIDByProfileID(ctx, profileID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrProfileNotFound
		}
		return "", fmt.Errorf("failed to resolve profile ID: %w", err)
	}

	return userUUID.String(), nil
}

// GetBasicProfileByUUID gets basic profile information by user UUID
func (s *ProfileService) GetBasicProfileByUUID(ctx context.Context, userUUID string) (*models.UserProfile, error) {
	parsedUUID, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user UUID format: %v", ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetBasicProfileByUUID(ctx, parsedUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get basic profile: %w", err)
	}

	return profile, nil
}

func (s *ProfileService) calculateAge(dateOfBirth time.Time) int {
	now := time.Now()
	age := now.Year() - dateOfBirth.Year()
	if now.Month() < dateOfBirth.Month() || (now.Month() == dateOfBirth.Month() && now.Day() < dateOfBirth.Day()) {
		age--
	}
	return age
}

func (s *ProfileService) GetDetailedProfileByID(ctx context.Context, profileID uint64) (*DetailedProfileData, error) {
	// Validate input
	if profileID == 0 {
		return nil, fmt.Errorf("%w: profile ID cannot be zero", ErrInvalidInput)
	}

	// Get the profile by ID
	profile, err := s.profileRepo.GetProfileByID(ctx, uint(profileID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil || profile.IsDeleted {
		return nil, ErrProfileNotFound
	}

	// Get partner preferences
	partnerPreferences, err := s.profileRepo.GetPartnerPreferences(ctx, profile.ID)
	if err != nil {
		s.logger.Warn("Failed to get partner preferences", "profileID", profileID, "error", err)
		// Don't fail the request if partner preferences are not found
	}

	// Get additional photos
	photos, err := s.profileRepo.GetUserPhotos(ctx, profile.UserID)
	if err != nil {
		s.logger.Warn("Failed to get user photos", "profileID", profileID, "error", err)
		photos = []*models.UserPhoto{} // Use empty slice if error
	}

	// Get intro video
	video, err := s.profileRepo.GetUserVideo(ctx, profile.UserID)
	if err != nil {
		s.logger.Warn("Failed to get user video", "profileID", profileID, "error", err)
		// Don't fail the request if video is not found
	}

	// Calculate age
	var age int
	if profile.DateOfBirth != nil {
		age = s.calculateAge(*profile.DateOfBirth)
	}

	// Build the detailed profile response
	detailedProfile := &DetailedProfileData{
		ID:                    profile.ID,
		IsBride:               profile.IsBride,
		FullName:              profile.FullName,
		DateOfBirth:           profile.DateOfBirth,
		HeightCM:              profile.HeightCM,
		PhysicallyChallenged:  profile.PhysicallyChallenged,
		Community:             profile.Community,
		MaritalStatus:         profile.MaritalStatus,
		Profession:            profile.Profession,
		ProfessionType:        profile.ProfessionType,
		HighestEducationLevel: profile.HighestEducationLevel,
		HomeDistrict:          profile.HomeDistrict,
		ProfilePictureURL:     profile.ProfilePictureURL,
		LastLogin:             profile.LastLogin,
		PartnerPreferences:    partnerPreferences,
		AdditionalPhotos:      photos,
		IntroVideo:            video,
		Age:                   age,
	}

	return detailedProfile, nil
}
