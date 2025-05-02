package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           uint           `gorm:"primaryKey"`
	UUID         string         `gorm:"type:uuid;default:gen_random_uuid();uniqueIndex"`
	Email        string         `gorm:"uniqueIndex"`
	Phone        string         `gorm:"uniqueIndex"`
	PasswordHash string         `gorm:"not null"`
	Verified     bool           `gorm:"default:false"`
	Role         string         `gorm:"default:'USER'"`
	PremiumUntil *time.Time     `gorm:"default:null"`
	LastLoginAt  *time.Time     `gorm:"default:null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates a UUID if not present
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.UUID == "" {
		u.UUID = uuid.New().String()
	}
	return nil
}
