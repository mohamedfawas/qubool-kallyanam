package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

// RegistrationRepo implements the registration repository interface
type RegistrationRepo struct {
	db *gorm.DB
}

// NewRegistrationRepository creates a new registration repository
func NewRegistrationRepository(db *gorm.DB) repositories.RegistrationRepository {
	return &RegistrationRepo{
		db: db,
	}
}

// CreatePendingRegistration stores a new pending registration
func (r *RegistrationRepo) CreatePendingRegistration(ctx context.Context, registration *models.PendingRegistration) error {
	// Store in database with context
	return r.db.WithContext(ctx).Create(registration).Error
}

// GetPendingRegistration finds a pending registration by field (email or phone)
func (r *RegistrationRepo) GetPendingRegistration(ctx context.Context, field string, value string) (*models.PendingRegistration, error) {
	var registration models.PendingRegistration
	result := r.db.WithContext(ctx).Where(field+" = ?", value).First(&registration)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &registration, nil
}

// DeletePendingRegistration removes a pending registration by ID
func (r *RegistrationRepo) DeletePendingRegistration(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&models.PendingRegistration{}, id).Error
}

// IsRegistered checks if a field (email or phone) is already registered in users table
func (r *RegistrationRepo) IsRegistered(ctx context.Context, field string, value string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("users").Where(field+" = ?", value).Count(&count).Error
	return count > 0, err
}
