package repositories

import (
	"context"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
)

type UserRepository interface {
	GetUser(ctx context.Context, field, value string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	SoftDeleteUser(ctx context.Context, userID string) error
	UpdateLastLogin(ctx context.Context, userID string) error
	UpdatePremiumUntil(ctx context.Context, userID string, premiumUntil time.Time) error
	IsRegistered(ctx context.Context, field, value string) (bool, error)
}
