package v1

import (
	"context"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
	userErrors "github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
)

type VideoHandler struct {
	videoService *services.VideoService
	jwtManager   *jwt.Manager
	logger       logging.Logger
}

func NewVideoHandler(
	videoService *services.VideoService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *VideoHandler {
	return &VideoHandler{
		videoService: videoService,
		jwtManager:   jwtManager,
		logger:       logger,
	}
}

// extractUserID is a helper method to extract user ID from incoming context metadata
func (h *VideoHandler) extractUserID(ctx context.Context) (string, error) {
	// Get metadata from context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "Missing metadata")
	}

	// Check for user ID in metadata (this is set by the gateway)
	userIDs := md.Get("user-id")
	if len(userIDs) > 0 && userIDs[0] != "" {
		return userIDs[0], nil
	}

	// As a fallback, check authorization header and extract from token
	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return "", status.Error(codes.Unauthenticated, "Authentication required")
	}

	tokenStr := strings.TrimPrefix(authHeader[0], "Bearer ")
	claims, err := h.jwtManager.ValidateToken(tokenStr)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "Invalid authentication")
	}

	userID := claims.UserID
	if userID == "" {
		return "", status.Error(codes.Unauthenticated, "User ID not found in token")
	}

	return userID, nil
}

// createTempFile creates a temporary file from the given data
func (h *VideoHandler) createTempFile(data []byte, filename string) (*os.File, error) {
	// Create a temporary file with a unique name
	ext := filepath.Ext(filename)
	prefix := "video_upload_"
	if len(filename) > 8 {
		prefix = filename[:8] + "_"
	}

	tempFile, err := os.CreateTemp("", prefix+"*"+ext)
	if err != nil {
		return nil, err
	}

	// Write data to the file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	// Reset file pointer to beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	return tempFile, nil
}

// validateVideoFile validates video file extension and size
func (h *VideoHandler) validateVideoFile(data []byte, filename string) error {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	validExt := false
	for _, allowedExt := range constants.AllowedVideoExtensions {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		return userErrors.ErrInvalidFileType
	}

	// Check file size
	if len(data) > constants.MaxVideoFileSize {
		return userErrors.ErrFileTooLarge
	}

	return nil
}

// UploadUserVideo handles uploading introduction video
func (h *VideoHandler) UploadUserVideo(ctx context.Context, req *userpb.UploadUserVideoRequest) (*userpb.UploadUserVideoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Validate input
	if len(req.GetVideoData()) == 0 {
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: "Video data is required",
			Error:   "empty video data",
		}, status.Error(codes.InvalidArgument, "Video data is required")
	}

	if req.GetFileName() == "" {
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: "Filename is required",
			Error:   "missing filename",
		}, status.Error(codes.InvalidArgument, "Filename is required")
	}

	// Validate file
	if err := h.validateVideoFile(req.GetVideoData(), req.GetFileName()); err != nil {
		h.logger.Error("File validation failed", "error", err, "userID", userID)
		var errMsg string
		switch {
		case errors.Is(err, userErrors.ErrInvalidFileType):
			errMsg = "Unsupported file type. Allowed types: mp4, mov, avi, mkv"
		case errors.Is(err, userErrors.ErrFileTooLarge):
			errMsg = "File size exceeds the maximum allowed size"
		default:
			errMsg = "Invalid file"
		}
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: errMsg,
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, errMsg)
	}

	// Create temporary file
	tempFile, err := h.createTempFile(req.GetVideoData(), req.GetFileName())
	if err != nil {
		h.logger.Error("Failed to create temp file", "error", err, "userID", userID)
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: "Failed to process video",
			Error:   "temp file creation failed",
		}, status.Error(codes.Internal, "Failed to process video")
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Create file header
	fileHeader := &multipart.FileHeader{
		Filename: req.GetFileName(),
		Size:     int64(len(req.GetVideoData())),
		Header:   make(map[string][]string),
	}
	if req.GetContentType() != "" {
		fileHeader.Header.Set("Content-Type", req.GetContentType())
	}

	// Reset file pointer
	tempFile.Seek(0, 0)

	// Upload video
	videoURL, err := h.videoService.UploadUserVideo(ctx, userID, fileHeader, tempFile)
	if err != nil {
		h.logger.Error("Failed to upload user video", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrVideoUploadFailed):
			errMsg = "Failed to upload video"
			statusCode = codes.Internal
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.UploadUserVideoResponse{
			Success: false,
			Message: "Failed to upload video",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UploadUserVideoResponse{
		Success:  true,
		Message:  "Video uploaded successfully",
		VideoUrl: videoURL,
	}, nil
}

// GetUserVideo retrieves the video for a user
func (h *VideoHandler) GetUserVideo(ctx context.Context, req *userpb.GetUserVideoRequest) (*userpb.GetUserVideoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetUserVideoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Get video
	video, err := h.videoService.GetUserVideo(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user video", "error", err, "userID", userID)
		return &userpb.GetUserVideoResponse{
			Success: false,
			Message: "Failed to retrieve video",
			Error:   "internal server error",
		}, status.Error(codes.Internal, "Failed to retrieve video")
	}

	if video == nil {
		return &userpb.GetUserVideoResponse{
			Success: true,
			Message: "No video found",
		}, nil
	}

	// Convert to proto format
	protoVideo := &userpb.UserVideoData{
		VideoUrl:  video.VideoURL,
		FileName:  video.FileName,
		FileSize:  video.FileSize,
		CreatedAt: timestamppb.New(video.CreatedAt),
	}

	if video.DurationSeconds != nil {
		protoVideo.DurationSeconds = int32(*video.DurationSeconds)
	}

	return &userpb.GetUserVideoResponse{
		Success: true,
		Message: "Video retrieved successfully",
		Video:   protoVideo,
	}, nil
}

// DeleteUserVideo deletes the user's introduction video
func (h *VideoHandler) DeleteUserVideo(ctx context.Context, req *userpb.DeleteUserVideoRequest) (*userpb.DeleteUserVideoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.DeleteUserVideoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Delete video
	err = h.videoService.DeleteUserVideo(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to delete user video", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrVideoNotFound):
			errMsg = "Video not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrVideoDeleteFailed):
			errMsg = "Failed to delete video"
			statusCode = codes.Internal
		default:
			errMsg = "Failed to delete video"
			statusCode = codes.Internal
		}
		return &userpb.DeleteUserVideoResponse{
			Success: false,
			Message: errMsg,
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.DeleteUserVideoResponse{
		Success: true,
		Message: "Video deleted successfully",
	}, nil
}
