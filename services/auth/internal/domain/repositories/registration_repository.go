package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
)

type RegistrationRepository interface {
	CreatePendingRegistration(ctx context.Context, registration *models.PendingRegistration) error
	GetPendingRegistration(ctx context.Context, field, value string) (*models.PendingRegistration, error)
	DeletePendingRegistration(ctx context.Context, id int) error
}
