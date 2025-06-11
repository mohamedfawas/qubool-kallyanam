package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

type ProfileRepository interface {
	CreateProfile(ctx context.Context, profile *models.UserProfile) error
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID, lastLogin time.Time) error
	UpdateEmail(ctx context.Context, userID uuid.UUID, email string) error
	ProfileExists(ctx context.Context, userID uuid.UUID) (bool, error)
	UpdateProfile(ctx context.Context, profile *models.UserProfile) error
	PatchProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error
	UpdateProfilePhoto(ctx context.Context, userID uuid.UUID, photoURL string) error
	RemoveProfilePhoto(ctx context.Context, userID uuid.UUID) error
	GetPartnerPreferences(ctx context.Context, userProfileID uint) (*models.PartnerPreferences, error)
	UpdatePartnerPreferences(ctx context.Context, prefs *models.PartnerPreferences) error
	GetProfileByID(ctx context.Context, id uint) (*models.UserProfile, error)
	SoftDeleteUserProfile(ctx context.Context, userID uuid.UUID) error
	SoftDeletePartnerPreferences(ctx context.Context, profileID uint) error
	GetUserUUIDByProfileID(ctx context.Context, profileID uint64) (uuid.UUID, error)
	GetBasicProfileByUUID(ctx context.Context, userUUID uuid.UUID) (*models.UserProfile, error)
	CreateUserPhoto(ctx context.Context, photo *models.UserPhoto) error
	GetUserPhotos(ctx context.Context, userID uuid.UUID) ([]*models.UserPhoto, error)
	DeleteUserPhoto(ctx context.Context, userID uuid.UUID, displayOrder int) error
	CountUserPhotos(ctx context.Context, userID uuid.UUID) (int, error)
	CreateUserVideo(ctx context.Context, video *models.UserVideo) error
	GetUserVideo(ctx context.Context, userID uuid.UUID) (*models.UserVideo, error)
	DeleteUserVideo(ctx context.Context, userID uuid.UUID) error
	HasUserVideo(ctx context.Context, userID uuid.UUID) (bool, error)
}
