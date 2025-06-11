package models

import (
	"time"

	"github.com/google/uuid"
)

// UserPhoto represents additional photos uploaded by users (excluding profile photos)
type UserPhoto struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index:idx_user_photos_user_id"`
	PhotoURL     string    `gorm:"size:500;not null;column:photo_url"`
	PhotoKey     string    `gorm:"size:500;not null;column:photo_key"` // S3 object key for deletion
	DisplayOrder int       `gorm:"not null;check:display_order>=1 AND display_order<=3;column:display_order"`
	CreatedAt    time.Time `gorm:"not null;default:now();column:created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:now();column:updated_at"`
}

// TableName returns the table name for UserPhoto
func (UserPhoto) TableName() string {
	return "user_photos"
}
