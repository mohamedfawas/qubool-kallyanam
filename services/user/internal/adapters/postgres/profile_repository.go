// user/internal/adapters/postgres/profile_repository.go
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
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
		Where("user_id = ?", userID).
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
