package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusSuccess  PaymentStatus = "success"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

type Payment struct {
	ID                uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID            uuid.UUID     `gorm:"type:uuid;not null;index"`
	RazorpayOrderID   string        `gorm:"size:255;unique;not null"`
	RazorpayPaymentID *string       `gorm:"size:255;default:null"`
	RazorpaySignature *string       `gorm:"size:255;default:null"`
	Amount            float64       `gorm:"type:decimal(10,2);not null"`
	Currency          string        `gorm:"size:3;not null;default:'INR'"`
	Status            PaymentStatus `gorm:"size:20;not null;default:'pending'"`
	PaymentMethod     *string       `gorm:"size:50;default:null"`
	CreatedAt         time.Time     `gorm:"not null"`
	UpdatedAt         time.Time     `gorm:"not null"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
