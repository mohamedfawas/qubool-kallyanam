package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string         `gorm:"size:255;not null;uniqueIndex:idx_users_email"`
	Phone        string         `gorm:"size:20;not null;uniqueIndex:idx_users_phone"`
	PasswordHash string         `gorm:"size:255;not null"`
	Verified     bool           `gorm:"not null;default:false"`
	PremiumUntil *time.Time     `gorm:"default:null"`
	LastLoginAt  *time.Time     `gorm:"column:last_login_at;default:null"`
	IsActive     bool           `gorm:"not null;default:true"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index;column:deleted_at"`
}

// BeforeCreate generates a UUID if not present
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// IsPremium checks if user has active premium subscription
func (u *User) IsPremium() bool {
	if u.PremiumUntil == nil {
		return false
	}
	return time.Now().Before(*u.PremiumUntil)
}
