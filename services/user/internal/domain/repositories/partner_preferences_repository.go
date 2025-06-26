package repositories

import (
	"context"

	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

type PartnerPreferencesRepository interface {
	GetPartnerPreferences(ctx context.Context, userProfileID uint) (*models.PartnerPreferences, error)
	UpdatePartnerPreferences(ctx context.Context, prefs *models.PartnerPreferences) error
	SoftDeletePartnerPreferences(ctx context.Context, profileID uint) error
}
