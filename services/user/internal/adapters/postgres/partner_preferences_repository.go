package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

// PartnerPreferencesRepo implements the partner preferences repository interface
type PartnerPreferencesRepo struct {
	db *gorm.DB
}

// NewPartnerPreferencesRepository creates a new partner preferences repository
func NewPartnerPreferencesRepository(db *gorm.DB) repositories.PartnerPreferencesRepository {
	return &PartnerPreferencesRepo{
		db: db,
	}
}

// GetPartnerPreferences retrieves partner preferences for a user profile
func (r *PartnerPreferencesRepo) GetPartnerPreferences(ctx context.Context, userProfileID uint) (*models.PartnerPreferences, error) {
	// Instead of directly loading into PartnerPreferences, use the WithArrays struct
	var prefsWithArrays models.PartnerPreferencesWithArrays
	err := r.db.WithContext(ctx).Where("user_profile_id = ? AND is_deleted = ?", userProfileID, false).First(&prefsWithArrays).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	// Convert the array fields to the correct types
	prefs := &models.PartnerPreferences{
		ID:                         prefsWithArrays.ID,
		UserProfileID:              prefsWithArrays.UserProfileID,
		MinAgeYears:                prefsWithArrays.MinAgeYears,
		MaxAgeYears:                prefsWithArrays.MaxAgeYears,
		MinHeightCM:                prefsWithArrays.MinHeightCM,
		MaxHeightCM:                prefsWithArrays.MaxHeightCM,
		AcceptPhysicallyChallenged: prefsWithArrays.AcceptPhysicallyChallenged,
		CreatedAt:                  prefsWithArrays.CreatedAt,
		UpdatedAt:                  prefsWithArrays.UpdatedAt,
		IsDeleted:                  prefsWithArrays.IsDeleted,
		DeletedAt:                  prefsWithArrays.DeletedAt,
	}

	// Convert string arrays to typed arrays
	prefs.PreferredCommunities = make([]models.Community, len(prefsWithArrays.PreferredCommunitiesArray))
	for i, v := range prefsWithArrays.PreferredCommunitiesArray {
		prefs.PreferredCommunities[i] = models.Community(v)
	}

	prefs.PreferredMaritalStatus = make([]models.MaritalStatus, len(prefsWithArrays.PreferredMaritalStatusArray))
	for i, v := range prefsWithArrays.PreferredMaritalStatusArray {
		prefs.PreferredMaritalStatus[i] = models.MaritalStatus(v)
	}

	prefs.PreferredProfessions = make([]models.Profession, len(prefsWithArrays.PreferredProfessionsArray))
	for i, v := range prefsWithArrays.PreferredProfessionsArray {
		prefs.PreferredProfessions[i] = models.Profession(v)
	}

	prefs.PreferredProfessionTypes = make([]models.ProfessionType, len(prefsWithArrays.PreferredProfessionTypesArray))
	for i, v := range prefsWithArrays.PreferredProfessionTypesArray {
		prefs.PreferredProfessionTypes[i] = models.ProfessionType(v)
	}

	prefs.PreferredEducationLevels = make([]models.EducationLevel, len(prefsWithArrays.PreferredEducationLevelsArray))
	for i, v := range prefsWithArrays.PreferredEducationLevelsArray {
		prefs.PreferredEducationLevels[i] = models.EducationLevel(v)
	}

	prefs.PreferredHomeDistricts = make([]models.HomeDistrict, len(prefsWithArrays.PreferredHomeDistrictsArray))
	for i, v := range prefsWithArrays.PreferredHomeDistrictsArray {
		prefs.PreferredHomeDistricts[i] = models.HomeDistrict(v)
	}

	return prefs, nil
}

// UpdatePartnerPreferences creates or updates partner preferences
func (r *PartnerPreferencesRepo) UpdatePartnerPreferences(ctx context.Context, prefs *models.PartnerPreferences) error {
	// Use the WithArrays struct for database operations
	prefsWithArrays := &models.PartnerPreferencesWithArrays{
		ID:                         prefs.ID,
		UserProfileID:              prefs.UserProfileID,
		MinAgeYears:                prefs.MinAgeYears,
		MaxAgeYears:                prefs.MaxAgeYears,
		MinHeightCM:                prefs.MinHeightCM,
		MaxHeightCM:                prefs.MaxHeightCM,
		AcceptPhysicallyChallenged: prefs.AcceptPhysicallyChallenged,
		CreatedAt:                  prefs.CreatedAt,
		UpdatedAt:                  prefs.UpdatedAt,
		IsDeleted:                  prefs.IsDeleted,
		DeletedAt:                  prefs.DeletedAt,
	}

	// Convert typed arrays to string arrays for PostgreSQL
	prefsWithArrays.PreferredCommunitiesArray = make([]string, len(prefs.PreferredCommunities))
	for i, v := range prefs.PreferredCommunities {
		prefsWithArrays.PreferredCommunitiesArray[i] = string(v)
	}

	prefsWithArrays.PreferredMaritalStatusArray = make([]string, len(prefs.PreferredMaritalStatus))
	for i, v := range prefs.PreferredMaritalStatus {
		prefsWithArrays.PreferredMaritalStatusArray[i] = string(v)
	}

	prefsWithArrays.PreferredProfessionsArray = make([]string, len(prefs.PreferredProfessions))
	for i, v := range prefs.PreferredProfessions {
		prefsWithArrays.PreferredProfessionsArray[i] = string(v)
	}

	prefsWithArrays.PreferredProfessionTypesArray = make([]string, len(prefs.PreferredProfessionTypes))
	for i, v := range prefs.PreferredProfessionTypes {
		prefsWithArrays.PreferredProfessionTypesArray[i] = string(v)
	}

	prefsWithArrays.PreferredEducationLevelsArray = make([]string, len(prefs.PreferredEducationLevels))
	for i, v := range prefs.PreferredEducationLevels {
		prefsWithArrays.PreferredEducationLevelsArray[i] = string(v)
	}

	prefsWithArrays.PreferredHomeDistrictsArray = make([]string, len(prefs.PreferredHomeDistricts))
	for i, v := range prefs.PreferredHomeDistricts {
		prefsWithArrays.PreferredHomeDistrictsArray[i] = string(v)
	}

	// Check if preferences already exist
	var existingPrefs models.PartnerPreferencesWithArrays
	err := r.db.WithContext(ctx).Where("user_profile_id = ? AND is_deleted = ?", prefs.UserProfileID, false).First(&existingPrefs).Error

	if err == gorm.ErrRecordNotFound {
		// Create new preferences
		return r.db.WithContext(ctx).Create(prefsWithArrays).Error
	} else if err != nil {
		return err
	} else {
		// Update existing preferences
		prefsWithArrays.ID = existingPrefs.ID
		prefsWithArrays.CreatedAt = existingPrefs.CreatedAt
		return r.db.WithContext(ctx).Save(prefsWithArrays).Error
	}
}

// SoftDeletePartnerPreferences soft deletes partner preferences
func (r *PartnerPreferencesRepo) SoftDeletePartnerPreferences(ctx context.Context, profileID uint) error {
	now := indianstandardtime.Now()
	return r.db.WithContext(ctx).Model(&models.PartnerPreferences{}).
		Where("user_profile_id = ?", profileID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
			"updated_at": now,
		}).Error
}
