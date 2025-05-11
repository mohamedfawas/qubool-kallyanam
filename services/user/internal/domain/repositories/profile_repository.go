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
	UpdateProfilePhoto(ctx context.Context, userID uuid.UUID, photoURL string) error
}
