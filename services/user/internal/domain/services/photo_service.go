package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/storage"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

var (
	ErrPhotoUploadFailed = errors.New("failed to upload photo")
)

// PhotoService handles photo management operations
type PhotoService struct {
	profileRepo  repositories.ProfileRepository
	photoStorage storage.PhotoStorage
	logger       logging.Logger
}

// NewPhotoService creates a new photo service
func NewPhotoService(
	profileRepo repositories.ProfileRepository,
	photoStorage storage.PhotoStorage,
	logger logging.Logger,
) *PhotoService {
	return &PhotoService{
		profileRepo:  profileRepo,
		photoStorage: photoStorage,
		logger:       logger,
	}
}

// UploadProfilePhoto uploads a profile photo for a user
func (s *PhotoService) UploadProfilePhoto(ctx context.Context, userID string, header *multipart.FileHeader, file io.Reader) (string, error) {
	// Validate and parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return "", ErrProfileNotFound
	}

	// Upload photo to storage
	photoURL, err := s.photoStorage.UploadProfilePhoto(ctx, userUUID, header, file)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPhotoUploadFailed, err)
	}

	// Update profile with new photo URL
	if err := s.profileRepo.UpdateProfilePhoto(ctx, userUUID, photoURL); err != nil {
		s.logger.Error("Failed to update profile photo URL", "error", err, "userID", userID)
		return "", fmt.Errorf("failed to update profile photo URL: %w", err)
	}

	s.logger.Info("Profile photo updated successfully", "userID", userID, "photoURL", photoURL)
	return photoURL, nil
}
