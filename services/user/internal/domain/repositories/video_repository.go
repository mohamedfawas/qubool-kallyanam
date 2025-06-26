package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

type VideoRepository interface {
	CreateUserVideo(ctx context.Context, video *models.UserVideo) error
	GetUserVideo(ctx context.Context, userID uuid.UUID) (*models.UserVideo, error)
	DeleteUserVideo(ctx context.Context, userID uuid.UUID) error
	HasUserVideo(ctx context.Context, userID uuid.UUID) (bool, error)
}
