package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
	"gorm.io/gorm"
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
	profileRepo            repositories.ProfileRepository
	partnerPreferencesRepo repositories.PartnerPreferencesRepository
	photoRepo              repositories.PhotoRepository
	videoRepo              repositories.VideoRepository
	logger                 logging.Logger
}

func NewProfileService(
	profileRepo repositories.ProfileRepository,
	partnerPreferencesRepo repositories.PartnerPreferencesRepository,
	photoRepo repositories.PhotoRepository,
	videoRepo repositories.VideoRepository,
	logger logging.Logger,
) *ProfileService {
	return &ProfileService{
		profileRepo:            profileRepo,
		partnerPreferencesRepo: partnerPreferencesRepo,
		photoRepo:              photoRepo,
		videoRepo:              videoRepo,
		logger:                 logger,
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
	profile := &models.UserProfile{
		UserID:                userUUID,
		Phone:                 phone,
		Email:                 email,
		IsBride:               false,
		LastLogin:             lastLogin,
		CreatedAt:             now,
		UpdatedAt:             now,
		Community:             constants.CommunityNotMentioned,
		MaritalStatus:         constants.MaritalNotMentioned,
		Profession:            constants.ProfessionNotMentioned,
		ProfessionType:        constants.ProfessionTypeNotMentioned,
		HighestEducationLevel: constants.EducationNotMentioned,
		HomeDistrict:          constants.DistrictNotMentioned,
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

	if err := s.partnerPreferencesRepo.SoftDeletePartnerPreferences(ctx, profile.ID); err != nil {
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
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Get existing profile
	existingProfile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if existingProfile == nil {
		return errors.ErrProfileNotFound
	}

	// Validate all fields
	if err := s.validateProfileFields(heightCM, community, maritalStatus, profession,
		professionType, educationLevel, homeDistrict, dateOfBirth); err != nil {
		return fmt.Errorf("%w: %v", errors.ErrValidation, err)
	}

	var dobTime *time.Time
	if dateOfBirth != "" {
		parsedTime, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: invalid date format: %v", errors.ErrInvalidInput, err)
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

func (s *ProfileService) validateProfileFields(heightCM *int, community string, maritalStatus string,
	profession string, professionType string, educationLevel string,
	homeDistrict string, dateOfBirth string) error {

	if heightCM != nil {
		if err := validation.ValidateHeight(heightCM); err != nil {
			return err
		}
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

	return nil
}

func (s *ProfileService) PatchProfile(ctx context.Context, userID string,
	isBride *bool, fullName *string, dateOfBirth string, heightCM *int,
	physicallyChallenged *bool, community *string, maritalStatus *string,
	profession *string, professionType *string, educationLevel *string,
	homeDistrict *string, clearDateOfBirth bool, clearHeightCM bool) error {

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return errors.ErrProfileNotFound
	}

	// Validate the fields that are being updated
	if heightCM != nil {
		if err := validation.ValidateHeight(heightCM); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if community != nil {
		if err := validation.ValidateCommunity(*community); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if maritalStatus != nil {
		if err := validation.ValidateMaritalStatus(*maritalStatus); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if profession != nil {
		if err := validation.ValidateProfession(*profession); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if professionType != nil {
		if err := validation.ValidateProfessionType(*professionType); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if educationLevel != nil {
		if err := validation.ValidateEducationLevel(*educationLevel); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	if homeDistrict != nil {
		if err := validation.ValidateHomeDistrict(*homeDistrict); err != nil {
			return fmt.Errorf("%w: %v", errors.ErrValidation, err)
		}
	}

	updates := make(map[string]interface{})

	if isBride != nil {
		updates["is_bride"] = *isBride
	}

	if fullName != nil {
		updates["full_name"] = *fullName
	}

	if dateOfBirth != "" && !clearDateOfBirth {
		parsedTime, err := time.Parse("2006-01-02", dateOfBirth)
		if err != nil {
			return fmt.Errorf("%w: invalid date format: %v", errors.ErrInvalidInput, err)
		}
		updates["date_of_birth"] = parsedTime
	}

	if clearDateOfBirth {
		updates["date_of_birth"] = nil
	}

	if heightCM != nil && !clearHeightCM {
		updates["height_cm"] = *heightCM
	}

	if clearHeightCM {
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

	if len(updates) == 0 {
		s.logger.Debug("No fields to update", "userID", userID)
		return nil
	}

	if err := s.profileRepo.PatchProfile(ctx, userUUID, updates); err != nil {
		return fmt.Errorf("failed to patch profile: %w", err)
	}

	s.logger.Info("Profile patched successfully", "userID", userID, "fieldsUpdated", len(updates))
	return nil
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving profile: %w", err)
	}

	if profile == nil {
		return nil, errors.ErrProfileNotFound
	}

	return profile, nil
}

// GetUserUUIDByProfileID resolves public profile ID to user UUID
func (s *ProfileService) GetUserUUIDByProfileID(ctx context.Context, profileID uint64) (string, error) {
	if profileID == 0 {
		return "", fmt.Errorf("%w: profile ID cannot be zero", errors.ErrInvalidInput)
	}

	userUUID, err := s.profileRepo.GetUserUUIDByProfileID(ctx, profileID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.ErrProfileNotFound
		}
		return "", fmt.Errorf("failed to resolve profile ID: %w", err)
	}

	return userUUID.String(), nil
}

// GetBasicProfileByUUID gets basic profile information by user UUID
func (s *ProfileService) GetBasicProfileByUUID(ctx context.Context, userUUID string) (*models.UserProfile, error) {
	parsedUUID, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user UUID format: %v", errors.ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetBasicProfileByUUID(ctx, parsedUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProfileNotFound
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
	if profileID == 0 {
		return nil, fmt.Errorf("%w: profile ID cannot be zero", errors.ErrInvalidInput)
	}

	profile, err := s.profileRepo.GetProfileByID(ctx, uint(profileID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	detailedData := &DetailedProfileData{
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
	}

	if profile.DateOfBirth != nil {
		detailedData.Age = s.calculateAge(*profile.DateOfBirth)
	}

	// Get partner preferences
	if preferences, err := s.partnerPreferencesRepo.GetPartnerPreferences(ctx, profile.ID); err == nil && preferences != nil {
		detailedData.PartnerPreferences = preferences
	}

	// Get additional photos
	if photos, err := s.photoRepo.GetUserPhotos(ctx, profile.UserID); err == nil {
		detailedData.AdditionalPhotos = photos
	}

	// Get intro video
	if video, err := s.videoRepo.GetUserVideo(ctx, profile.UserID); err == nil && video != nil {
		detailedData.IntroVideo = video
	}

	return detailedData, nil
}

// GetDetailedProfileByUUID gets comprehensive profile information by user UUID (for admin)
func (s *ProfileService) GetDetailedProfileByUUID(ctx context.Context, userUUID string) (*DetailedProfileData, error) {
	parsedUUID, err := uuid.Parse(userUUID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user UUID format: %v", errors.ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, parsedUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return nil, errors.ErrProfileNotFound
	}

	detailedData := &DetailedProfileData{
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
	}

	if profile.DateOfBirth != nil {
		detailedData.Age = s.calculateAge(*profile.DateOfBirth)
	}

	// Get partner preferences
	if preferences, err := s.partnerPreferencesRepo.GetPartnerPreferences(ctx, profile.ID); err == nil && preferences != nil {
		detailedData.PartnerPreferences = preferences
	}

	// Get additional photos
	if photos, err := s.photoRepo.GetUserPhotos(ctx, profile.UserID); err == nil {
		detailedData.AdditionalPhotos = photos
	}

	// Get intro video
	if video, err := s.videoRepo.GetUserVideo(ctx, profile.UserID); err == nil && video != nil {
		detailedData.IntroVideo = video
	}

	return detailedData, nil
}
