package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

type PhotoRepository interface {
	CreateUserPhoto(ctx context.Context, photo *models.UserPhoto) error
	GetUserPhotos(ctx context.Context, userID uuid.UUID) ([]*models.UserPhoto, error)
	DeleteUserPhoto(ctx context.Context, userID uuid.UUID, displayOrder int) error
	CountUserPhotos(ctx context.Context, userID uuid.UUID) (int, error)
}
