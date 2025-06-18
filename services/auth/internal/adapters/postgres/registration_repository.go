package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

type RegistrationRepo struct {
	db *gorm.DB
}

func NewRegistrationRepository(db *gorm.DB) repositories.RegistrationRepository {
	return &RegistrationRepo{
		db: db,
	}
}

func (r *RegistrationRepo) CreatePendingRegistration(ctx context.Context, registration *models.PendingRegistration) error {
	return r.db.WithContext(ctx).Create(registration).Error
}

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

func (r *RegistrationRepo) DeletePendingRegistration(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&models.PendingRegistration{}, id).Error
}
