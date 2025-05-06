package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
)

type AdminRepository interface {
	// GetAdminByEmail returns an admin by email or nil if not found
	GetAdminByEmail(ctx context.Context, email string) (*models.Admin, error)

	// CreateAdmin creates a new admin
	CreateAdmin(ctx context.Context, admin *models.Admin) error

	// UpdateAdmin updates an existing admin
	UpdateAdmin(ctx context.Context, admin *models.Admin) error

	// CheckAdminExists checks if any admin exists in the system
	CheckAdminExists(ctx context.Context) (bool, error)
}
