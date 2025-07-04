package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
)

type PartnerPreferencesService struct {
	profileRepo            repositories.ProfileRepository
	partnerPreferencesRepo repositories.PartnerPreferencesRepository
	logger                 logging.Logger
}

func NewPartnerPreferencesService(
	profileRepo repositories.ProfileRepository,
	partnerPreferencesRepo repositories.PartnerPreferencesRepository,
	logger logging.Logger,
) *PartnerPreferencesService {
	return &PartnerPreferencesService{
		profileRepo:            profileRepo,
		partnerPreferencesRepo: partnerPreferencesRepo,
		logger:                 logger,
	}
}

// ValidatePartnerPreferences validates partner preferences data
func (s *PartnerPreferencesService) ValidatePartnerPreferences(
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
) error {
	// Validate age range
	if minAgeYears != nil && maxAgeYears != nil {
		if *minAgeYears < constants.MinAge || *maxAgeYears > constants.MaxAge || *minAgeYears > *maxAgeYears {
			return errors.ErrInvalidAgeRange
		}
	} else if minAgeYears != nil && *minAgeYears < constants.MinAge {
		return errors.ErrInvalidAgeRange
	} else if maxAgeYears != nil && *maxAgeYears > constants.MaxAge {
		return errors.ErrInvalidAgeRange
	}

	// Validate height range
	if minHeightCM != nil && maxHeightCM != nil {
		if *minHeightCM < constants.MinHeight || *maxHeightCM > constants.MaxHeight || *minHeightCM > *maxHeightCM {
			return errors.ErrInvalidHeightRange
		}
	} else if minHeightCM != nil && *minHeightCM < constants.MinHeight {
		return errors.ErrInvalidHeightRange
	} else if maxHeightCM != nil && *maxHeightCM > constants.MaxHeight {
		return errors.ErrInvalidHeightRange
	}

	// Validate preferred communities
	for _, community := range preferredCommunities {
		if err := validation.ValidateCommunity(community); err != nil {
			return fmt.Errorf("invalid preferred community '%s': %w", community, err)
		}
	}

	// Validate preferred marital statuses
	for _, status := range preferredMaritalStatus {
		if err := validation.ValidateMaritalStatus(status); err != nil {
			return fmt.Errorf("invalid preferred marital status '%s': %w", status, err)
		}
	}

	// Validate preferred professions
	for _, profession := range preferredProfessions {
		if err := validation.ValidateProfession(profession); err != nil {
			return fmt.Errorf("invalid preferred profession '%s': %w", profession, err)
		}
	}

	// Validate preferred profession types
	for _, profType := range preferredProfessionTypes {
		if err := validation.ValidateProfessionType(profType); err != nil {
			return fmt.Errorf("invalid preferred profession type '%s': %w", profType, err)
		}
	}

	// Validate preferred education levels
	for _, level := range preferredEducationLevels {
		if err := validation.ValidateEducationLevel(level); err != nil {
			return fmt.Errorf("invalid preferred education level '%s': %w", level, err)
		}
	}

	// Validate preferred home districts
	for _, district := range preferredHomeDistricts {
		if err := validation.ValidateHomeDistrict(district); err != nil {
			return fmt.Errorf("invalid preferred home district '%s': %w", district, err)
		}
	}

	return nil
}

// GetPartnerPreferences retrieves partner preferences for a user
func (s *PartnerPreferencesService) GetPartnerPreferences(ctx context.Context, userID string) (*models.PartnerPreferences, error) {
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

	prefs, err := s.partnerPreferencesRepo.GetPartnerPreferences(ctx, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving partner preferences: %w", err)
	}

	return prefs, nil
}

// UpdatePartnerPreferences creates or updates partner preferences for a user
func (s *PartnerPreferencesService) UpdatePartnerPreferences(
	ctx context.Context,
	userID string,
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	acceptPhysicallyChallenged bool,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Get user profile first
	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if profile == nil {
		return errors.ErrProfileNotFound
	}

	// Validate partner preferences
	err = s.ValidatePartnerPreferences(
		minAgeYears,
		maxAgeYears,
		minHeightCM,
		maxHeightCM,
		preferredCommunities,
		preferredMaritalStatus,
		preferredProfessions,
		preferredProfessionTypes,
		preferredEducationLevels,
		preferredHomeDistricts,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrValidation, err)
	}

	// Convert string arrays to typed arrays
	typedCommunities := make([]models.Community, 0, len(preferredCommunities))
	typedMaritalStatus := make([]models.MaritalStatus, 0, len(preferredMaritalStatus))
	typedProfessions := make([]models.Profession, 0, len(preferredProfessions))
	typedProfessionTypes := make([]models.ProfessionType, 0, len(preferredProfessionTypes))
	typedEducationLevels := make([]models.EducationLevel, 0, len(preferredEducationLevels))
	typedHomeDistricts := make([]models.HomeDistrict, 0, len(preferredHomeDistricts))

	for _, c := range preferredCommunities {
		typedCommunities = append(typedCommunities, models.Community(c))
	}
	for _, ms := range preferredMaritalStatus {
		typedMaritalStatus = append(typedMaritalStatus, models.MaritalStatus(ms))
	}
	for _, p := range preferredProfessions {
		typedProfessions = append(typedProfessions, models.Profession(p))
	}
	for _, pt := range preferredProfessionTypes {
		typedProfessionTypes = append(typedProfessionTypes, models.ProfessionType(pt))
	}
	for _, el := range preferredEducationLevels {
		typedEducationLevels = append(typedEducationLevels, models.EducationLevel(el))
	}
	for _, hd := range preferredHomeDistricts {
		typedHomeDistricts = append(typedHomeDistricts, models.HomeDistrict(hd))
	}

	// Create or update partner preferences
	now := indianstandardtime.Now()
	partnerPrefs := &models.PartnerPreferences{
		UserProfileID:              profile.ID,
		MinAgeYears:                minAgeYears,
		MaxAgeYears:                maxAgeYears,
		MinHeightCM:                minHeightCM,
		MaxHeightCM:                maxHeightCM,
		AcceptPhysicallyChallenged: acceptPhysicallyChallenged,
		PreferredCommunities:       typedCommunities,
		PreferredMaritalStatus:     typedMaritalStatus,
		PreferredProfessions:       typedProfessions,
		PreferredProfessionTypes:   typedProfessionTypes,
		PreferredEducationLevels:   typedEducationLevels,
		PreferredHomeDistricts:     typedHomeDistricts,
		UpdatedAt:                  now,
	}

	if err := s.partnerPreferencesRepo.UpdatePartnerPreferences(ctx, partnerPrefs); err != nil {
		s.logger.Error("Failed to update partner preferences", "userID", userID, "error", err)
		return fmt.Errorf("failed to update partner preferences: %w", err)
	}

	s.logger.Info("Partner preferences updated successfully", "userID", userID)
	return nil
}

// PatchPartnerPreferences updates specific partner preference fields
func (s *PartnerPreferencesService) PatchPartnerPreferences(
	ctx context.Context,
	userID string,
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	acceptPhysicallyChallenged *bool,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
	clearPreferredCommunities bool,
	clearPreferredMaritalStatus bool,
	clearPreferredProfessions bool,
	clearPreferredProfessionTypes bool,
	clearPreferredEducationLevels bool,
	clearPreferredHomeDistricts bool,
) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if profile == nil {
		return errors.ErrProfileNotFound
	}

	// Get current preferences
	currentPrefs, err := s.partnerPreferencesRepo.GetPartnerPreferences(ctx, profile.ID)
	if err != nil {
		return fmt.Errorf("error retrieving partner preferences: %w", err)
	}

	// Create new preferences with defaults from current preferences
	now := indianstandardtime.Now()
	newPrefs := &models.PartnerPreferences{
		UserProfileID: profile.ID,
		UpdatedAt:     now,
	}

	// Copy existing values or use new ones if provided
	if currentPrefs != nil {
		newPrefs.ID = currentPrefs.ID
		newPrefs.CreatedAt = currentPrefs.CreatedAt
		newPrefs.MinAgeYears = currentPrefs.MinAgeYears
		newPrefs.MaxAgeYears = currentPrefs.MaxAgeYears
		newPrefs.MinHeightCM = currentPrefs.MinHeightCM
		newPrefs.MaxHeightCM = currentPrefs.MaxHeightCM
		newPrefs.AcceptPhysicallyChallenged = currentPrefs.AcceptPhysicallyChallenged
		newPrefs.PreferredCommunities = currentPrefs.PreferredCommunities
		newPrefs.PreferredMaritalStatus = currentPrefs.PreferredMaritalStatus
		newPrefs.PreferredProfessions = currentPrefs.PreferredProfessions
		newPrefs.PreferredProfessionTypes = currentPrefs.PreferredProfessionTypes
		newPrefs.PreferredEducationLevels = currentPrefs.PreferredEducationLevels
		newPrefs.PreferredHomeDistricts = currentPrefs.PreferredHomeDistricts
	} else {
		// Set defaults for new preferences
		newPrefs.AcceptPhysicallyChallenged = true
		newPrefs.CreatedAt = now
	}

	// Update fields if provided
	if minAgeYears != nil {
		newPrefs.MinAgeYears = minAgeYears
	}
	if maxAgeYears != nil {
		newPrefs.MaxAgeYears = maxAgeYears
	}
	if minHeightCM != nil {
		newPrefs.MinHeightCM = minHeightCM
	}
	if maxHeightCM != nil {
		newPrefs.MaxHeightCM = maxHeightCM
	}
	if acceptPhysicallyChallenged != nil {
		newPrefs.AcceptPhysicallyChallenged = *acceptPhysicallyChallenged
	}

	// Handle array fields
	if clearPreferredCommunities {
		newPrefs.PreferredCommunities = []models.Community{}
	} else if len(preferredCommunities) > 0 {
		typedCommunities := make([]models.Community, len(preferredCommunities))
		for i, c := range preferredCommunities {
			typedCommunities[i] = models.Community(c)
		}
		newPrefs.PreferredCommunities = typedCommunities
	}

	if clearPreferredMaritalStatus {
		newPrefs.PreferredMaritalStatus = []models.MaritalStatus{}
	} else if len(preferredMaritalStatus) > 0 {
		typedMaritalStatus := make([]models.MaritalStatus, len(preferredMaritalStatus))
		for i, ms := range preferredMaritalStatus {
			typedMaritalStatus[i] = models.MaritalStatus(ms)
		}
		newPrefs.PreferredMaritalStatus = typedMaritalStatus
	}

	if clearPreferredProfessions {
		newPrefs.PreferredProfessions = []models.Profession{}
	} else if len(preferredProfessions) > 0 {
		typedProfessions := make([]models.Profession, len(preferredProfessions))
		for i, p := range preferredProfessions {
			typedProfessions[i] = models.Profession(p)
		}
		newPrefs.PreferredProfessions = typedProfessions
	}

	if clearPreferredProfessionTypes {
		newPrefs.PreferredProfessionTypes = []models.ProfessionType{}
	} else if len(preferredProfessionTypes) > 0 {
		typedProfessionTypes := make([]models.ProfessionType, len(preferredProfessionTypes))
		for i, pt := range preferredProfessionTypes {
			typedProfessionTypes[i] = models.ProfessionType(pt)
		}
		newPrefs.PreferredProfessionTypes = typedProfessionTypes
	}

	if clearPreferredEducationLevels {
		newPrefs.PreferredEducationLevels = []models.EducationLevel{}
	} else if len(preferredEducationLevels) > 0 {
		typedEducationLevels := make([]models.EducationLevel, len(preferredEducationLevels))
		for i, el := range preferredEducationLevels {
			typedEducationLevels[i] = models.EducationLevel(el)
		}
		newPrefs.PreferredEducationLevels = typedEducationLevels
	}

	if clearPreferredHomeDistricts {
		newPrefs.PreferredHomeDistricts = []models.HomeDistrict{}
	} else if len(preferredHomeDistricts) > 0 {
		typedHomeDistricts := make([]models.HomeDistrict, len(preferredHomeDistricts))
		for i, hd := range preferredHomeDistricts {
			typedHomeDistricts[i] = models.HomeDistrict(hd)
		}
		newPrefs.PreferredHomeDistricts = typedHomeDistricts
	}

	// Validate the updated preferences
	communities := make([]string, len(newPrefs.PreferredCommunities))
	for i, c := range newPrefs.PreferredCommunities {
		communities[i] = string(c)
	}

	maritalStatuses := make([]string, len(newPrefs.PreferredMaritalStatus))
	for i, ms := range newPrefs.PreferredMaritalStatus {
		maritalStatuses[i] = string(ms)
	}

	professions := make([]string, len(newPrefs.PreferredProfessions))
	for i, p := range newPrefs.PreferredProfessions {
		professions[i] = string(p)
	}

	professionTypes := make([]string, len(newPrefs.PreferredProfessionTypes))
	for i, pt := range newPrefs.PreferredProfessionTypes {
		professionTypes[i] = string(pt)
	}

	educationLevels := make([]string, len(newPrefs.PreferredEducationLevels))
	for i, el := range newPrefs.PreferredEducationLevels {
		educationLevels[i] = string(el)
	}

	homeDistricts := make([]string, len(newPrefs.PreferredHomeDistricts))
	for i, hd := range newPrefs.PreferredHomeDistricts {
		homeDistricts[i] = string(hd)
	}

	if err := s.ValidatePartnerPreferences(
		newPrefs.MinAgeYears,
		newPrefs.MaxAgeYears,
		newPrefs.MinHeightCM,
		newPrefs.MaxHeightCM,
		communities,
		maritalStatuses,
		professions,
		professionTypes,
		educationLevels,
		homeDistricts,
	); err != nil {
		return fmt.Errorf("%w: %v", errors.ErrValidation, err)
	}

	// Update preferences
	if err := s.partnerPreferencesRepo.UpdatePartnerPreferences(ctx, newPrefs); err != nil {
		s.logger.Error("Failed to patch partner preferences", "userID", userID, "error", err)
		return fmt.Errorf("failed to patch partner preferences: %w", err)
	}

	s.logger.Info("Partner preferences patched successfully", "userID", userID)
	return nil
}
