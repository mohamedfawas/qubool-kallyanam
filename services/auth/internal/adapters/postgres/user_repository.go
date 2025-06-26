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

// GetUsers implements admin user listing with filtering (reuses existing patterns)
func (r *UserRepo) GetUsers(ctx context.Context, params repositories.GetUsersParams) ([]*models.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.User{})

	// Apply status filter (reuse existing is_active pattern)
	switch params.Status {
	case "active":
		query = query.Where("is_active = ? AND deleted_at IS NULL", true)
	case "inactive":
		query = query.Where("is_active = ?", false)
	case "soft_deleted":
		query = query.Where("deleted_at IS NOT NULL")
	case "all":
		// No status filter
	default:
		// Default to active users (same as existing GetUser pattern)
		query = query.Where("is_active = ? AND deleted_at IS NULL", true)
	}

	// Apply verification filter
	if params.VerifiedOnly != nil {
		query = query.Where("verified = ?", *params.VerifiedOnly)
	}

	// Apply premium filter
	if params.PremiumOnly != nil {
		if *params.PremiumOnly {
			query = query.Where("premium_until IS NOT NULL AND premium_until > NOW()")
		} else {
			query = query.Where("premium_until IS NULL OR premium_until <= NOW()")
		}
	}

	// Apply date filters
	if params.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *params.CreatedAfter)
	}
	if params.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *params.CreatedBefore)
	}
	if params.LastLoginAfter != nil {
		query = query.Where("last_login_at >= ?", *params.LastLoginAfter)
	}
	if params.LastLoginBefore != nil {
		query = query.Where("last_login_at <= ?", *params.LastLoginBefore)
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting (default patterns)
	sortBy := "created_at"
	if params.SortBy != "" {
		switch params.SortBy {
		case "created_at", "last_login_at", "email", "updated_at":
			sortBy = params.SortBy
		}
	}

	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	// Execute query
	var users []*models.User
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
