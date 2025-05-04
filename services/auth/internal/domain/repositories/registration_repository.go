package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
)

// RegistrationRepository defines methods for working with registrations
type RegistrationRepository interface {
	// CreatePendingRegistration creates a new pending registration
	CreatePendingRegistration(ctx context.Context, registration *models.PendingRegistration) error

	// GetPendingRegistration finds a pending registration by field (email or phone)
	GetPendingRegistration(ctx context.Context, field string, value string) (*models.PendingRegistration, error)

	// DeletePendingRegistration removes a pending registration by ID
	DeletePendingRegistration(ctx context.Context, id int) error

	// IsRegistered checks if a field (email or phone) is already registered in users table
	IsRegistered(ctx context.Context, field string, value string) (bool, error)
}
