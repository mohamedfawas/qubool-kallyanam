// user/internal/adapters/postgres/profile_repository.go
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

// ProfileRepo implements the profile repository interface
type ProfileRepo struct {
	db *gorm.DB
}

// NewProfileRepository creates a new profile repository
func NewProfileRepository(db *gorm.DB) repositories.ProfileRepository {
	return &ProfileRepo{
		db: db,
	}
}

// CreateProfile stores a new user profile with basic information
func (r *ProfileRepo) CreateProfile(ctx context.Context, profile *models.UserProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

// GetProfileByUserID retrieves a profile by user ID
func (r *ProfileRepo) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_deleted = ?", userID, false).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil when not found
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *ProfileRepo) UpdateLastLogin(ctx context.Context, userID uuid.UUID, lastLogin time.Time) error {
	return r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", userID).
		Update("last_login", lastLogin).
		Error
}

// ProfileExists checks if a profile exists for the given user ID
func (r *ProfileRepo) ProfileExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).
		Error
	return count > 0, err
}

func (r *ProfileRepo) UpdateProfile(ctx context.Context, profile *models.UserProfile) error {
	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", profile.UserID).
		Updates(map[string]interface{}{
			"is_bride":                profile.IsBride,
			"full_name":               profile.FullName,
			"date_of_birth":           profile.DateOfBirth,
			"height_cm":               profile.HeightCM,
			"physically_challenged":   profile.PhysicallyChallenged,
			"community":               profile.Community,
			"marital_status":          profile.MaritalStatus,
			"profession":              profile.Profession,
			"profession_type":         profile.ProfessionType,
			"highest_education_level": profile.HighestEducationLevel,
			"home_district":           profile.HomeDistrict,
			"updated_at":              profile.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		// Profile doesn't exist
		return gorm.ErrRecordNotFound
	}

	return nil
}

// UpdateProfilePhoto updates the profile picture URL for a user
func (r *ProfileRepo) UpdateProfilePhoto(ctx context.Context, userID uuid.UUID, photoURL string) error {
	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"profile_picture_url": photoURL,
			"updated_at":          time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		// Profile doesn't exist
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *ProfileRepo) RemoveProfilePhoto(ctx context.Context, userID uuid.UUID) error {
	now := indianstandardtime.Now()
	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"profile_picture_url": nil,
			"updated_at":          now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ProfileRepo) PatchProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	// Add updated_at to every update
	updates["updated_at"] = indianstandardtime.Now()

	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *ProfileRepo) GetPartnerPreferences(ctx context.Context, userProfileID uint) (*models.PartnerPreferences, error) {
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

func (r *ProfileRepo) UpdatePartnerPreferences(ctx context.Context, prefs *models.PartnerPreferences) error {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PartnerPreferences{}).
		Where("user_profile_id = ?", prefs.UserProfileID).
		Count(&count).Error
	if err != nil {
		return err
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := indianstandardtime.Now()
	prefs.UpdatedAt = now

	// Convert typed arrays to string arrays
	communitiesStr := make([]string, len(prefs.PreferredCommunities))
	for i, v := range prefs.PreferredCommunities {
		communitiesStr[i] = string(v)
	}

	maritalStatusStr := make([]string, len(prefs.PreferredMaritalStatus))
	for i, v := range prefs.PreferredMaritalStatus {
		maritalStatusStr[i] = string(v)
	}

	professionsStr := make([]string, len(prefs.PreferredProfessions))
	for i, v := range prefs.PreferredProfessions {
		professionsStr[i] = string(v)
	}

	professionTypesStr := make([]string, len(prefs.PreferredProfessionTypes))
	for i, v := range prefs.PreferredProfessionTypes {
		professionTypesStr[i] = string(v)
	}

	educationLevelsStr := make([]string, len(prefs.PreferredEducationLevels))
	for i, v := range prefs.PreferredEducationLevels {
		educationLevelsStr[i] = string(v)
	}

	homeDistrictsStr := make([]string, len(prefs.PreferredHomeDistricts))
	for i, v := range prefs.PreferredHomeDistricts {
		homeDistrictsStr[i] = string(v)
	}

	if count > 0 {
		// Update existing record
		sql := `UPDATE partner_preferences SET 
                min_age_years = ?, max_age_years = ?, 
                min_height_cm = ?, max_height_cm = ?, 
                accept_physically_challenged = ?,
                preferred_communities = ?, 
                preferred_marital_status = ?, 
                preferred_professions = ?, 
                preferred_profession_types = ?, 
                preferred_education_levels = ?, 
                preferred_home_districts = ?,
                updated_at = ?
                WHERE user_profile_id = ?`

		err = tx.Exec(sql,
			prefs.MinAgeYears, prefs.MaxAgeYears,
			prefs.MinHeightCM, prefs.MaxHeightCM,
			prefs.AcceptPhysicallyChallenged,
			pq.Array(communitiesStr),
			pq.Array(maritalStatusStr),
			pq.Array(professionsStr),
			pq.Array(professionTypesStr),
			pq.Array(educationLevelsStr),
			pq.Array(homeDistrictsStr),
			now,
			prefs.UserProfileID).Error
	} else {
		// Insert new record
		sql := `INSERT INTO partner_preferences (
                user_profile_id, min_age_years, max_age_years, 
                min_height_cm, max_height_cm, accept_physically_challenged,
                preferred_communities, preferred_marital_status, 
                preferred_professions, preferred_profession_types, 
                preferred_education_levels, preferred_home_districts,
                created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		err = tx.Exec(sql,
			prefs.UserProfileID, prefs.MinAgeYears, prefs.MaxAgeYears,
			prefs.MinHeightCM, prefs.MaxHeightCM, prefs.AcceptPhysicallyChallenged,
			pq.Array(communitiesStr),
			pq.Array(maritalStatusStr),
			pq.Array(professionsStr),
			pq.Array(professionTypesStr),
			pq.Array(educationLevelsStr),
			pq.Array(homeDistrictsStr),
			now, now).Error
	}

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *ProfileRepo) GetProfileByID(ctx context.Context, id uint) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepo) SoftDeleteUserProfile(ctx context.Context, userID uuid.UUID) error {
	now := indianstandardtime.Now()
	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
			"updated_at": now,
		})

	return result.Error
}

func (r *ProfileRepo) SoftDeletePartnerPreferences(ctx context.Context, profileID uint) error {
	now := indianstandardtime.Now()
	result := r.db.WithContext(ctx).Model(&models.PartnerPreferences{}).
		Where("user_profile_id = ? AND is_deleted = ?", profileID, false).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
			"updated_at": now,
		})

	return result.Error
}

// GetUserUUIDByProfileID resolves a public profile ID to user UUID
func (r *ProfileRepo) GetUserUUIDByProfileID(ctx context.Context, profileID uint64) (uuid.UUID, error) {
	var profile models.UserProfile
	err := r.db.WithContext(ctx).
		Select("user_id").
		Where("id = ? AND is_deleted = ?", profileID, false).
		First(&profile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return uuid.Nil, err
		}
		return uuid.Nil, err
	}

	return profile.UserID, nil
}

// GetBasicProfileByUUID gets basic profile information by user UUID
func (r *ProfileRepo) GetBasicProfileByUUID(ctx context.Context, userUUID uuid.UUID) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := r.db.WithContext(ctx).
		Select("id, full_name, profile_picture_url, is_deleted").
		Where("user_id = ? AND is_deleted = ?", userUUID, false).
		First(&profile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, err
	}

	return &profile, nil
}

func (r *ProfileRepo) UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error {
	result := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"email":      email,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// CreateUserPhoto creates a new user photo record
func (r *ProfileRepo) CreateUserPhoto(ctx context.Context, photo *models.UserPhoto) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

// GetUserPhotos retrieves all photos for a user, ordered by display_order
func (r *ProfileRepo) GetUserPhotos(ctx context.Context, userID uuid.UUID) ([]*models.UserPhoto, error) {
	var photos []*models.UserPhoto
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("display_order ASC").
		Find(&photos).Error
	return photos, err
}

// DeleteUserPhoto hard deletes a user photo
func (r *ProfileRepo) DeleteUserPhoto(ctx context.Context, userID uuid.UUID, displayOrder int) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND display_order = ?", userID, displayOrder).
		Delete(&models.UserPhoto{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// CountUserPhotos counts the number of photos for a user
func (r *ProfileRepo) CountUserPhotos(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserPhoto{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return int(count), err
}

// CreateUserVideo creates a new user video record
func (r *ProfileRepo) CreateUserVideo(ctx context.Context, video *models.UserVideo) error {
	return r.db.WithContext(ctx).Create(video).Error
}

// GetUserVideo retrieves the video for a user
func (r *ProfileRepo) GetUserVideo(ctx context.Context, userID uuid.UUID) (*models.UserVideo, error) {
	var video models.UserVideo
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&video).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &video, nil
}

// DeleteUserVideo hard deletes a user video
func (r *ProfileRepo) DeleteUserVideo(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.UserVideo{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// HasUserVideo checks if user has a video
func (r *ProfileRepo) HasUserVideo(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserVideo{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}
