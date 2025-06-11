package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/storage"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

var (
	ErrPhotoUploadFailed = errors.New("failed to upload photo")
	ErrPhotoDeleteFailed = errors.New("failed to delete photo")
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

func (s *PhotoService) DeleteProfilePhoto(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Check if profile exists
	exists, err := s.profileRepo.ProfileExists(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error checking profile existence: %w", err)
	}
	if !exists {
		return ErrProfileNotFound
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
		return fmt.Errorf("%w: %v", ErrPhotoDeleteFailed, err)
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

	// Validate display order
	if displayOrder < 1 || displayOrder > 3 {
		return "", fmt.Errorf("%w: display order must be between 1 and 3", ErrInvalidInput)
	}

	// Check if user already has 3 photos
	photoCount, err := s.profileRepo.CountUserPhotos(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error counting user photos: %w", err)
	}
	if photoCount >= 3 {
		return "", fmt.Errorf("%w: maximum of 3 additional photos allowed", ErrInvalidInput)
	}

	// Upload photo to storage
	photoURL, photoKey, err := s.photoStorage.UploadUserPhoto(ctx, userUUID, header, file, displayOrder)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPhotoUploadFailed, err)
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

	if err := s.profileRepo.CreateUserPhoto(ctx, photo); err != nil {
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
		return nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	photos, err := s.profileRepo.GetUserPhotos(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user photos: %w", err)
	}

	return photos, nil
}

// DeleteUserPhoto deletes a specific user photo by display order
func (s *PhotoService) DeleteUserPhoto(ctx context.Context, userID string, displayOrder int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Validate display order
	if displayOrder < 1 || displayOrder > 3 {
		return fmt.Errorf("%w: display order must be between 1 and 3", ErrInvalidInput)
	}

	// Get the photo to delete to retrieve the storage key
	photos, err := s.profileRepo.GetUserPhotos(ctx, userUUID)
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
		return fmt.Errorf("%w: %v", ErrPhotoDeleteFailed, err)
	}

	// Delete from database
	if err := s.profileRepo.DeleteUserPhoto(ctx, userUUID, displayOrder); err != nil {
		s.logger.Error("Failed to delete photo from database", "error", err, "userID", userID, "displayOrder", displayOrder)
		return fmt.Errorf("failed to delete photo record: %w", err)
	}

	s.logger.Info("User photo deleted successfully", "userID", userID, "displayOrder", displayOrder)
	return nil
}

// UploadUserVideo uploads an introduction video for a user
func (s *PhotoService) UploadUserVideo(ctx context.Context, userID string, header *multipart.FileHeader, file io.Reader) (string, error) {
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

	// Check if user already has a video
	hasVideo, err := s.profileRepo.HasUserVideo(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error checking existing video: %w", err)
	}
	if hasVideo {
		return "", fmt.Errorf("%w: user already has an introduction video. Please delete the existing one first", ErrInvalidInput)
	}

	// Upload video to storage
	videoURL, videoKey, err := s.photoStorage.UploadUserVideo(ctx, userUUID, header, file)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPhotoUploadFailed, err)
	}

	// Create video record in database
	video := &models.UserVideo{
		UserID:          userUUID,
		VideoURL:        videoURL,
		VideoKey:        videoKey,
		FileName:        header.Filename,
		FileSize:        header.Size,
		DurationSeconds: nil, // Could be populated later with video analysis
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.profileRepo.CreateUserVideo(ctx, video); err != nil {
		// If database save fails, attempt to delete from storage
		if deleteErr := s.photoStorage.DeleteUserVideo(ctx, videoKey); deleteErr != nil {
			s.logger.Error("Failed to cleanup video from storage after database error",
				"error", deleteErr, "videoKey", videoKey)
		}
		return "", fmt.Errorf("failed to save video record: %w", err)
	}

	s.logger.Info("User video uploaded successfully", "userID", userID, "videoURL", videoURL)
	return videoURL, nil
}

// GetUserVideo retrieves the video for a user
func (s *PhotoService) GetUserVideo(ctx context.Context, userID string) (*models.UserVideo, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	video, err := s.profileRepo.GetUserVideo(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user video: %w", err)
	}

	return video, nil
}

// DeleteUserVideo deletes the user's introduction video
func (s *PhotoService) DeleteUserVideo(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Get the video to delete to retrieve the storage key
	video, err := s.profileRepo.GetUserVideo(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving user video: %w", err)
	}

	if video == nil {
		return fmt.Errorf("video not found for user")
	}

	// Delete from storage first
	if err := s.photoStorage.DeleteUserVideo(ctx, video.VideoKey); err != nil {
		s.logger.Error("Failed to delete video from storage", "error", err, "videoKey", video.VideoKey)
		return fmt.Errorf("%w: %v", ErrPhotoDeleteFailed, err)
	}

	// Delete from database
	if err := s.profileRepo.DeleteUserVideo(ctx, userUUID); err != nil {
		s.logger.Error("Failed to delete video from database", "error", err, "userID", userID)
		return fmt.Errorf("failed to delete video record: %w", err)
	}

	s.logger.Info("User video deleted successfully", "userID", userID)
	return nil
}
