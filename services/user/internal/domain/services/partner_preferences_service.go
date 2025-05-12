package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

var (
	ErrInvalidAgeRange            = errors.New("invalid age range: min age must be at least 18, max age must be at most 80, and min age must not exceed max age")
	ErrInvalidHeightRange         = errors.New("invalid height range: min height must be at least 130cm, max height must be at most 220cm, and min height must not exceed max height")
	ErrPartnerPreferencesNotFound = errors.New("partner preferences not found")
)

// ValidatePartnerPreferences validates partner preferences
func (s *ProfileService) ValidatePartnerPreferences(
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
		if *minAgeYears < 18 || *maxAgeYears > 80 || *minAgeYears > *maxAgeYears {
			return ErrInvalidAgeRange
		}
	} else if minAgeYears != nil && *minAgeYears < 18 {
		return ErrInvalidAgeRange
	} else if maxAgeYears != nil && *maxAgeYears > 80 {
		return ErrInvalidAgeRange
	}

	// Validate height range
	if minHeightCM != nil && maxHeightCM != nil {
		if *minHeightCM < 130 || *maxHeightCM > 220 || *minHeightCM > *maxHeightCM {
			return ErrInvalidHeightRange
		}
	} else if minHeightCM != nil && *minHeightCM < 130 {
		return ErrInvalidHeightRange
	} else if maxHeightCM != nil && *maxHeightCM > 220 {
		return ErrInvalidHeightRange
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
func (s *ProfileService) GetPartnerPreferences(ctx context.Context, userID string) (*models.PartnerPreferences, error) {
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

	prefs, err := s.profileRepo.GetPartnerPreferences(ctx, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving partner preferences: %w", err)
	}

	return prefs, nil
}

// UpdatePartnerPreferences updates partner preferences for a user
func (s *ProfileService) UpdatePartnerPreferences(
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
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Get user profile first
	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
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
		return fmt.Errorf("%w: %v", ErrValidation, err)
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
		UpdatedAt:                  indianstandardtime.Now(),
	}

	if err := s.profileRepo.UpdatePartnerPreferences(ctx, partnerPrefs); err != nil {
		return fmt.Errorf("failed to update partner preferences: %w", err)
	}

	s.logger.Info("Partner preferences updated successfully", "userID", userID)
	return nil
}

func (s *ProfileService) PatchPartnerPreferences(
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
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}
	if profile == nil {
		return ErrProfileNotFound
	}

	// Get current preferences
	currentPrefs, err := s.profileRepo.GetPartnerPreferences(ctx, profile.ID)
	if err != nil {
		return fmt.Errorf("error retrieving partner preferences: %w", err)
	}

	// Create new preferences with defaults from current preferences
	newPrefs := &models.PartnerPreferences{
		UserProfileID: profile.ID,
		UpdatedAt:     indianstandardtime.Now(),
	}

	// Copy existing values or use new ones if provided
	if currentPrefs != nil {
		newPrefs.MinAgeYears = currentPrefs.MinAgeYears
		newPrefs.MaxAgeYears = currentPrefs.MaxAgeYears
		newPrefs.MinHeightCM = currentPrefs.MinHeightCM
		newPrefs.MaxHeightCM = currentPrefs.MaxHeightCM
		newPrefs.AcceptPhysicallyChallenged = currentPrefs.AcceptPhysicallyChallenged

		if !clearPreferredCommunities {
			newPrefs.PreferredCommunities = currentPrefs.PreferredCommunities
		}
		if !clearPreferredMaritalStatus {
			newPrefs.PreferredMaritalStatus = currentPrefs.PreferredMaritalStatus
		}
		if !clearPreferredProfessions {
			newPrefs.PreferredProfessions = currentPrefs.PreferredProfessions
		}
		if !clearPreferredProfessionTypes {
			newPrefs.PreferredProfessionTypes = currentPrefs.PreferredProfessionTypes
		}
		if !clearPreferredEducationLevels {
			newPrefs.PreferredEducationLevels = currentPrefs.PreferredEducationLevels
		}
		if !clearPreferredHomeDistricts {
			newPrefs.PreferredHomeDistricts = currentPrefs.PreferredHomeDistricts
		}
	}

	// Apply updates if provided
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

	// Process array fields if provided
	typedCommunities := []models.Community{}
	typedMaritalStatus := []models.MaritalStatus{}
	typedProfessions := []models.Profession{}
	typedProfessionTypes := []models.ProfessionType{}
	typedEducationLevels := []models.EducationLevel{}
	typedHomeDistricts := []models.HomeDistrict{}

	if len(preferredCommunities) > 0 {
		for _, c := range preferredCommunities {
			typedCommunities = append(typedCommunities, models.Community(c))
		}
		newPrefs.PreferredCommunities = typedCommunities
	}
	if len(preferredMaritalStatus) > 0 {
		for _, ms := range preferredMaritalStatus {
			typedMaritalStatus = append(typedMaritalStatus, models.MaritalStatus(ms))
		}
		newPrefs.PreferredMaritalStatus = typedMaritalStatus
	}
	if len(preferredProfessions) > 0 {
		for _, p := range preferredProfessions {
			typedProfessions = append(typedProfessions, models.Profession(p))
		}
		newPrefs.PreferredProfessions = typedProfessions
	}
	if len(preferredProfessionTypes) > 0 {
		for _, pt := range preferredProfessionTypes {
			typedProfessionTypes = append(typedProfessionTypes, models.ProfessionType(pt))
		}
		newPrefs.PreferredProfessionTypes = typedProfessionTypes
	}
	if len(preferredEducationLevels) > 0 {
		for _, el := range preferredEducationLevels {
			typedEducationLevels = append(typedEducationLevels, models.EducationLevel(el))
		}
		newPrefs.PreferredEducationLevels = typedEducationLevels
	}
	if len(preferredHomeDistricts) > 0 {
		for _, hd := range preferredHomeDistricts {
			typedHomeDistricts = append(typedHomeDistricts, models.HomeDistrict(hd))
		}
		newPrefs.PreferredHomeDistricts = typedHomeDistricts
	}

	// Validate the updated preferences
	if err := s.ValidatePartnerPreferences(
		newPrefs.MinAgeYears,
		newPrefs.MaxAgeYears,
		newPrefs.MinHeightCM,
		newPrefs.MaxHeightCM,
		preferredCommunities,
		preferredMaritalStatus,
		preferredProfessions,
		preferredProfessionTypes,
		preferredEducationLevels,
		preferredHomeDistricts,
	); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// Update the preferences in the database
	if err := s.profileRepo.UpdatePartnerPreferences(ctx, newPrefs); err != nil {
		return fmt.Errorf("failed to update partner preferences: %w", err)
	}

	s.logger.Info("Partner preferences patched successfully", "userID", userID)
	return nil
}
