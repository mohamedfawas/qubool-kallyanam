package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionStatus string

const (
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

type Subscription struct {
	ID        uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID          `gorm:"type:uuid;not null;index"`
	PlanID    string             `gorm:"size:50;not null;default:'premium_365'"`
	Status    SubscriptionStatus `gorm:"size:20;not null;default:'pending'"`
	StartDate *time.Time         `gorm:"default:null"`
	EndDate   *time.Time         `gorm:"default:null"`
	Amount    float64            `gorm:"type:decimal(10,2);not null"`
	Currency  string             `gorm:"size:3;not null;default:'INR'"`
	PaymentID *uuid.UUID         `gorm:"type:uuid;default:null"`
	CreatedAt time.Time          `gorm:"not null"`
	UpdatedAt time.Time          `gorm:"not null"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (s *Subscription) IsActive() bool {
	if s.Status != SubscriptionStatusActive {
		return false
	}
	if s.EndDate == nil {
		return false
	}
	return time.Now().Before(*s.EndDate)
}

type SubscriptionPlan struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	DurationDays int     `json:"duration_days"`
	Amount       float64 `json:"amount"` // in rupees
	Currency     string  `json:"currency"`
}

var DefaultPlans = map[string]SubscriptionPlan{
	"premium_365": {
		ID:           "premium_365",
		Name:         "Premium Membership",
		DurationDays: 365,
		Amount:       1000.0,
		Currency:     "INR",
	},
}
