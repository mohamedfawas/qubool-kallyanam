package models

import (
	"time"
)

// PendingRegistration represents a user registration in progress
type PendingRegistration struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex"`
	Phone        string    `json:"phone" gorm:"uniqueIndex"`
	PasswordHash string    `json:"-"` // Never expose password hash
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TableName specifies table name for GORM
func (PendingRegistration) TableName() string {
	return "pending_registrations"
}

// Registration represents the data needed for a new registration
type Registration struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required,phone"`
	Password string `json:"password" validate:"required,min=8"`
}
