package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

// ProfileRepository defines operations on user profiles
type ProfileRepository interface {
	CreateProfile(ctx context.Context, profile *models.UserProfile) error
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID, lastLogin time.Time) error
	ProfileExists(ctx context.Context, userID uuid.UUID) (bool, error)
	UpdateProfile(ctx context.Context, profile *models.UserProfile) error
	PatchProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error
	UpdateProfilePhoto(ctx context.Context, userID uuid.UUID, photoURL string) error
	RemoveProfilePhoto(ctx context.Context, userID uuid.UUID) error
	GetPartnerPreferences(ctx context.Context, userProfileID uint) (*models.PartnerPreferences, error)
	UpdatePartnerPreferences(ctx context.Context, prefs *models.PartnerPreferences) error
}
