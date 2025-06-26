package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

// VideoRepo implements the video repository interface
type VideoRepo struct {
	db *gorm.DB
}

// NewVideoRepository creates a new video repository
func NewVideoRepository(db *gorm.DB) repositories.VideoRepository {
	return &VideoRepo{
		db: db,
	}
}

// CreateUserVideo creates a new user video record
func (r *VideoRepo) CreateUserVideo(ctx context.Context, video *models.UserVideo) error {
	return r.db.WithContext(ctx).Create(video).Error
}

// GetUserVideo retrieves a user's video
func (r *VideoRepo) GetUserVideo(ctx context.Context, userID uuid.UUID) (*models.UserVideo, error) {
	var video models.UserVideo
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&video).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil when not found
		}
		return nil, err
	}

	return &video, nil
}

// DeleteUserVideo deletes a user's video
func (r *VideoRepo) DeleteUserVideo(ctx context.Context, userID uuid.UUID) error {
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

// HasUserVideo checks if a user has a video
func (r *VideoRepo) HasUserVideo(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserVideo{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count > 0, err
}
