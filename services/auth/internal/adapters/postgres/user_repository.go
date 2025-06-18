package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repositories.UserRepository {
	return &UserRepo{
		db: db,
	}
}

func (r *UserRepo) GetUser(ctx context.Context, field string, value string) (*models.User, error) {
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

func (r *UserRepo) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepo) SoftDeleteUser(ctx context.Context, userID string) error {
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

func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID string) error {
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

func (r *UserRepo) UpdatePremiumUntil(ctx context.Context, userID string, premiumUntil time.Time) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	now := indianstandardtime.Now()
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"premium_until": premiumUntil,
			"updated_at":    now,
		}).Error
}

func (r *UserRepo) IsRegistered(ctx context.Context, field, value string) (bool, error) {
	user, err := r.GetUser(ctx, field, value)
	if err != nil {
		return false, err
	}
	return user != nil, nil
}
