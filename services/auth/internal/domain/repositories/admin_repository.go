package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
)

type AdminRepository interface {
	GetAdminByEmail(ctx context.Context, email string) (*models.Admin, error)
	CreateAdmin(ctx context.Context, admin *models.Admin) error
	UpdateAdmin(ctx context.Context, admin *models.Admin) error
	CheckAdminExists(ctx context.Context) (bool, error)
}
