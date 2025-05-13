package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
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
func (r *RegistrationRepo) IsRegistered(ctx context.Context, field, value string) (bool, error) {
	user, err := r.GetUser(ctx, field, value)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}

// GetUser retrieves a user by the specified field (email or phone) , avoids soft deleted users
func (r *RegistrationRepo) GetUser(ctx context.Context, field string, value string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).Where(field+" = ?", value).Where("is_active = ?", true).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// CreateUser creates a new user in the database
func (r *RegistrationRepo) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *RegistrationRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	now := indianstandardtime.Now()
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
}

func (r *RegistrationRepo) SoftDeleteUser(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	now := indianstandardtime.Now()
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": now,
			"deleted_at": now,
		}).Error
}
