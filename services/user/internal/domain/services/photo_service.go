package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/storage"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
)

// PhotoService handles photo management operations
type PhotoService struct {
	profileRepo  repositories.ProfileRepository
	photoRepo    repositories.PhotoRepository
	photoStorage storage.PhotoStorage
	logger       logging.Logger
}

// NewPhotoService creates a new photo service
func NewPhotoService(
	profileRepo repositories.ProfileRepository,
	photoRepo repositories.PhotoRepository,
	photoStorage storage.PhotoStorage,
	logger logging.Logger,
) *PhotoService {
	return &PhotoService{
		profileRepo:  profileRepo,
		photoRepo:    photoRepo,
		photoStorage: photoStorage,
		logger:       logger,
	}
}

// UploadProfilePhoto uploads a profile photo for a user
func (s *PhotoService) UploadProfilePhoto(ctx context.Context, userID string, header *multipart.FileHeader, file io.Reader) (string, error) {
	// Validate and parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return "", errors.ErrProfileNotFound
	}

	// Validate file size
	if header.Size > constants.MaxFileSize {
		return "", fmt.Errorf("%w: file size exceeds maximum limit of %d bytes", errors.ErrInvalidInput, constants.MaxFileSize)
	}

	// Upload photo to storage
	photoURL, err := s.photoStorage.UploadProfilePhoto(ctx, userUUID, header, file)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errors.ErrPhotoUploadFailed, err)
	}

	// Update profile with new photo URL
	if err := s.profileRepo.UpdateProfilePhoto(ctx, userUUID, photoURL); err != nil {
		s.logger.Error("Failed to update profile photo URL", "error", err, "userID", userID)
		return "", fmt.Errorf("failed to update profile photo URL: %w", err)
	}

	s.logger.Info("Profile photo updated successfully", "userID", userID, "photoURL", photoURL)
	return photoURL, nil
}

// DeleteProfilePhoto deletes a user's profile photo
func (s *PhotoService) DeleteProfilePhoto(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return errors.ErrProfileNotFound
	}

	// Get profile to check if it has a photo
	profile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving profile: %w", err)
	}

	// If profile doesn't have a photo, return early as nothing to delete
	if profile.ProfilePictureURL == nil || *profile.ProfilePictureURL == "" {
		s.logger.Info("No profile photo to delete", "userID", userID)
		return nil
	}

	// Delete photo from storage
	if err := s.photoStorage.DeleteProfilePhoto(ctx, userUUID); err != nil {
		s.logger.Error("Failed to delete profile photo from storage", "error", err, "userID", userID)
		return fmt.Errorf("%w: %v", errors.ErrPhotoDeleteFailed, err)
	}

	// Update profile in database to remove photo URL
	if err := s.profileRepo.RemoveProfilePhoto(ctx, userUUID); err != nil {
		s.logger.Error("Failed to remove profile photo URL", "error", err, "userID", userID)
		return fmt.Errorf("failed to remove profile photo URL: %w", err)
	}

	s.logger.Info("Profile photo deleted successfully", "userID", userID)
	return nil
}

// UploadUserPhoto uploads an additional photo for a user
func (s *PhotoService) UploadUserPhoto(ctx context.Context, userID string, header *multipart.FileHeader, file io.Reader, displayOrder int) (string, error) {
	// Validate and parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return "", errors.ErrProfileNotFound
	}

	// Validate display order
	if displayOrder < constants.MinDisplayOrder || displayOrder > constants.MaxDisplayOrder {
		return "", fmt.Errorf("%w: display order must be between %d and %d", errors.ErrInvalidInput, constants.MinDisplayOrder, constants.MaxDisplayOrder)
	}

	// Validate file size
	if header.Size > constants.MaxFileSize {
		return "", fmt.Errorf("%w: file size exceeds maximum limit of %d bytes", errors.ErrInvalidInput, constants.MaxFileSize)
	}

	// Check if user already has maximum photos
	photoCount, err := s.photoRepo.CountUserPhotos(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error counting user photos: %w", err)
	}
	if photoCount >= constants.MaxAdditionalPhotos {
		return "", fmt.Errorf("%w: maximum of %d additional photos allowed", errors.ErrInvalidInput, constants.MaxAdditionalPhotos)
	}

	// Upload photo to storage
	photoURL, photoKey, err := s.photoStorage.UploadUserPhoto(ctx, userUUID, header, file, displayOrder)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errors.ErrPhotoUploadFailed, err)
	}

	// Create photo record in database
	photo := &models.UserPhoto{
		UserID:       userUUID,
		PhotoURL:     photoURL,
		PhotoKey:     photoKey,
		DisplayOrder: displayOrder,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.photoRepo.CreateUserPhoto(ctx, photo); err != nil {
		// If database save fails, attempt to delete from storage
		if deleteErr := s.photoStorage.DeleteUserPhoto(ctx, photoKey); deleteErr != nil {
			s.logger.Error("Failed to cleanup photo from storage after database error",
				"error", deleteErr, "photoKey", photoKey)
		}
		return "", fmt.Errorf("failed to save photo record: %w", err)
	}

	s.logger.Info("User photo uploaded successfully", "userID", userID, "displayOrder", displayOrder, "photoURL", photoURL)
	return photoURL, nil
}

// GetUserPhotos retrieves all photos for a user
func (s *PhotoService) GetUserPhotos(ctx context.Context, userID string) ([]*models.UserPhoto, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	photos, err := s.photoRepo.GetUserPhotos(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user photos: %w", err)
	}

	return photos, nil
}

// DeleteUserPhoto deletes a specific user photo by display order
func (s *PhotoService) DeleteUserPhoto(ctx context.Context, userID string, displayOrder int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	// Validate display order
	if displayOrder < constants.MinDisplayOrder || displayOrder > constants.MaxDisplayOrder {
		return fmt.Errorf("%w: display order must be between %d and %d", errors.ErrInvalidInput, constants.MinDisplayOrder, constants.MaxDisplayOrder)
	}

	// Get the photo to delete to retrieve the storage key
	photos, err := s.photoRepo.GetUserPhotos(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving user photos: %w", err)
	}

	var photoToDelete *models.UserPhoto
	for _, photo := range photos {
		if photo.DisplayOrder == displayOrder {
			photoToDelete = photo
			break
		}
	}

	if photoToDelete == nil {
		return fmt.Errorf("photo not found at display order %d", displayOrder)
	}

	// Delete from storage first
	if err := s.photoStorage.DeleteUserPhoto(ctx, photoToDelete.PhotoKey); err != nil {
		s.logger.Error("Failed to delete photo from storage", "error", err, "photoKey", photoToDelete.PhotoKey)
		return fmt.Errorf("%w: %v", errors.ErrPhotoDeleteFailed, err)
	}

	// Delete from database
	if err := s.photoRepo.DeleteUserPhoto(ctx, userUUID, displayOrder); err != nil {
		s.logger.Error("Failed to delete photo from database", "error", err, "userID", userID, "displayOrder", displayOrder)
		return fmt.Errorf("failed to delete photo record: %w", err)
	}

	s.logger.Info("User photo deleted successfully", "userID", userID, "displayOrder", displayOrder)
	return nil
}

// CountUserPhotos returns the number of photos a user has
func (s *PhotoService) CountUserPhotos(ctx context.Context, userID string) (int, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	count, err := s.photoRepo.CountUserPhotos(ctx, userUUID)
	if err != nil {
		return 0, fmt.Errorf("error counting user photos: %w", err)
	}

	return count, nil
}
