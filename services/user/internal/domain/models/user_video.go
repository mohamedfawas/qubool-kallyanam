package models

import (
	"time"

	"github.com/google/uuid"
)

// UserVideo represents introduction videos uploaded by users
type UserVideo struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex;column:user_id"`
	VideoURL        string    `gorm:"size:500;not null;column:video_url"`
	VideoKey        string    `gorm:"size:500;not null;column:video_key"` // S3 object key for deletion
	FileName        string    `gorm:"size:255;not null;column:file_name"`
	FileSize        int64     `gorm:"not null;column:file_size"` // Size in bytes
	DurationSeconds *int      `gorm:"column:duration_seconds"`   // Video duration
	CreatedAt       time.Time `gorm:"not null;default:now();column:created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:now();column:updated_at"`
}

// TableName returns the table name for UserVideo
func (UserVideo) TableName() string {
	return "user_videos"
}
