package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

// PhotoRepo implements the photo repository interface
type PhotoRepo struct {
	db *gorm.DB
}

// NewPhotoRepository creates a new photo repository
func NewPhotoRepository(db *gorm.DB) repositories.PhotoRepository {
	return &PhotoRepo{
		db: db,
	}
}

// CreateUserPhoto creates a new user photo record
func (r *PhotoRepo) CreateUserPhoto(ctx context.Context, photo *models.UserPhoto) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

// GetUserPhotos retrieves all photos for a user
func (r *PhotoRepo) GetUserPhotos(ctx context.Context, userID uuid.UUID) ([]*models.UserPhoto, error) {
	var photos []*models.UserPhoto
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("display_order ASC").
		Find(&photos).Error
	if err != nil {
		return nil, err
	}
	return photos, nil
}

// DeleteUserPhoto deletes a user photo by display order
func (r *PhotoRepo) DeleteUserPhoto(ctx context.Context, userID uuid.UUID, displayOrder int) error {
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
func (r *PhotoRepo) CountUserPhotos(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserPhoto{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return int(count), err
}
