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

// VideoService handles video management operations
type VideoService struct {
	profileRepo  repositories.ProfileRepository
	videoRepo    repositories.VideoRepository
	photoStorage storage.PhotoStorage // Note: Using PhotoStorage as it handles both photos and videos
	logger       logging.Logger
}

// NewVideoService creates a new video service
func NewVideoService(
	profileRepo repositories.ProfileRepository,
	videoRepo repositories.VideoRepository,
	photoStorage storage.PhotoStorage,
	logger logging.Logger,
) *VideoService {
	return &VideoService{
		profileRepo:  profileRepo,
		videoRepo:    videoRepo,
		photoStorage: photoStorage,
		logger:       logger,
	}
}

// UploadUserVideo uploads an introduction video for a user
func (s *VideoService) UploadUserVideo(ctx context.Context, userID string, header *multipart.FileHeader, file io.Reader) (string, error) {
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

	// Check if user already has a video
	hasVideo, err := s.videoRepo.HasUserVideo(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("error checking existing video: %w", err)
	}
	if hasVideo {
		return "", fmt.Errorf("%w: user already has an introduction video. Please delete the existing one first", errors.ErrInvalidInput)
	}

	// Upload video to storage
	videoURL, videoKey, err := s.photoStorage.UploadUserVideo(ctx, userUUID, header, file)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errors.ErrPhotoUploadFailed, err)
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

	if err := s.videoRepo.CreateUserVideo(ctx, video); err != nil {
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
func (s *VideoService) GetUserVideo(ctx context.Context, userID string) (*models.UserVideo, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	video, err := s.videoRepo.GetUserVideo(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user video: %w", err)
	}

	return video, nil
}

// DeleteUserVideo deletes the user's introduction video
func (s *VideoService) DeleteUserVideo(ctx context.Context, userID string) error {
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

	// Get the video to delete to retrieve the storage key
	video, err := s.videoRepo.GetUserVideo(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error retrieving user video: %w", err)
	}

	if video == nil {
		s.logger.Info("No video to delete", "userID", userID)
		return nil
	}

	// Delete from storage first
	if err := s.photoStorage.DeleteUserVideo(ctx, video.VideoKey); err != nil {
		s.logger.Error("Failed to delete video from storage", "error", err, "videoKey", video.VideoKey)
		return fmt.Errorf("%w: %v", errors.ErrPhotoDeleteFailed, err)
	}

	// Delete from database
	if err := s.videoRepo.DeleteUserVideo(ctx, userUUID); err != nil {
		s.logger.Error("Failed to delete video from database", "error", err, "userID", userID)
		return fmt.Errorf("failed to delete video record: %w", err)
	}

	s.logger.Info("User video deleted successfully", "userID", userID)
	return nil
}

// HasUserVideo checks if a user has an introduction video
func (s *VideoService) HasUserVideo(ctx context.Context, userID string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("%w: invalid user ID format: %v", errors.ErrInvalidInput, err)
	}

	hasVideo, err := s.videoRepo.HasUserVideo(ctx, userUUID)
	if err != nil {
		return false, fmt.Errorf("error checking user video: %w", err)
	}

	return hasVideo, nil
}
