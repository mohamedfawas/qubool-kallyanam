package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

type AdminRepo struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) repositories.AdminRepository {
	return &AdminRepo{
		db: db,
	}
}

func (r *AdminRepo) GetAdminByEmail(ctx context.Context, email string) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&admin)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &admin, nil
}

func (r *AdminRepo) CreateAdmin(ctx context.Context, admin *models.Admin) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *AdminRepo) UpdateAdmin(ctx context.Context, admin *models.Admin) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

func (r *AdminRepo) CheckAdminExists(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Admin{}).Count(&count).Error
	return count > 0, err
}
