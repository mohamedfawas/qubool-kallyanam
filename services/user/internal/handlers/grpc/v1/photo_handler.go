package v1

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
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

type PhotoHandler struct {
	photoService *services.PhotoService
	jwtManager   *jwt.Manager
	logger       logging.Logger
}

func NewPhotoHandler(
	photoService *services.PhotoService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *PhotoHandler {
	return &PhotoHandler{
		photoService: photoService,
		jwtManager:   jwtManager,
		logger:       logger,
	}
}

// extractUserID is a helper method to extract user ID from incoming context metadata
func (h *PhotoHandler) extractUserID(ctx context.Context) (string, error) {
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
func (h *PhotoHandler) createTempFile(data []byte, filename string) (*os.File, error) {
	// Create a temporary file with a unique name
	ext := filepath.Ext(filename)
	prefix := "upload_"
	if len(filename) > 8 {
		prefix = filename[:8]
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

// validateImageFile validates image file extension and size
func (h *PhotoHandler) validateImageFile(data []byte, filename string) error {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	validExt := false
	for _, allowedExt := range constants.AllowedImageExtensions {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		return userErrors.ErrInvalidFileType
	}

	// Check file size
	if len(data) > constants.MaxImageFileSize {
		return userErrors.ErrFileTooLarge
	}

	return nil
}

// UploadProfilePhoto handles the profile photo upload
func (h *PhotoHandler) UploadProfilePhoto(ctx context.Context, req *userpb.UploadProfilePhotoRequest) (*userpb.UploadProfilePhotoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Validate input
	if len(req.GetPhotoData()) == 0 {
		h.logger.Error("Empty photo data", "userID", userID)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Photo data is required",
			Error:   "empty photo data",
		}, status.Error(codes.InvalidArgument, "Photo data is required")
	}

	if req.GetFileName() == "" {
		h.logger.Error("Missing filename", "userID", userID)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Filename is required",
			Error:   "missing filename",
		}, status.Error(codes.InvalidArgument, "Filename is required")
	}

	// Validate file
	if err := h.validateImageFile(req.GetPhotoData(), req.GetFileName()); err != nil {
		h.logger.Error("File validation failed", "error", err, "userID", userID)
		var errMsg string
		switch {
		case errors.Is(err, userErrors.ErrInvalidFileType):
			errMsg = "Unsupported file type. Allowed types: jpg, jpeg, png, gif, webp"
		case errors.Is(err, userErrors.ErrFileTooLarge):
			errMsg = "File size exceeds the maximum allowed size of 5MB"
		default:
			errMsg = "Invalid file"
		}
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: errMsg,
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, errMsg)
	}

	// Create temporary file
	tempFile, err := h.createTempFile(req.GetPhotoData(), req.GetFileName())
	if err != nil {
		h.logger.Error("Failed to create temporary file", "error", err, "userID", userID)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Failed to process uploaded file",
			Error:   "temporary file creation failed",
		}, status.Error(codes.Internal, "Failed to process upload")
	}
	defer os.Remove(tempFile.Name()) // Clean up when done
	defer tempFile.Close()

	// Content type
	contentType := req.GetContentType()
	if contentType == "" {
		// Try to detect content type
		contentType = http.DetectContentType(req.GetPhotoData())
	}

	// Create a multipart file header
	header := &multipart.FileHeader{
		Filename: req.GetFileName(),
		Size:     int64(len(req.GetPhotoData())),
	}

	// Upload the photo
	photoURL, err := h.photoService.UploadProfilePhoto(ctx, userID, header, tempFile)
	if err != nil {
		h.logger.Error("Failed to upload profile photo", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrPhotoUploadFailed):
			errMsg = "Failed to upload photo"
			statusCode = codes.Internal
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Failed to upload profile photo",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UploadProfilePhotoResponse{
		Success:  true,
		Message:  "Profile photo uploaded successfully",
		PhotoUrl: photoURL,
	}, nil
}

// DeleteProfilePhoto deletes the user's profile photo
func (h *PhotoHandler) DeleteProfilePhoto(ctx context.Context, req *userpb.DeleteProfilePhotoRequest) (*userpb.DeleteProfilePhotoResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.DeleteProfilePhotoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	err = h.photoService.DeleteProfilePhoto(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to delete profile photo", "error", err, "userID", userID)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrPhotoDeleteFailed):
			errMsg = "Failed to delete photo"
			statusCode = codes.Internal
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}

		return &userpb.DeleteProfilePhotoResponse{
			Success: false,
			Message: "Failed to delete profile photo",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.DeleteProfilePhotoResponse{
		Success: true,
		Message: "Profile photo deleted successfully",
	}, nil
}

// UploadUserPhoto handles uploading additional photos
func (h *PhotoHandler) UploadUserPhoto(ctx context.Context, req *userpb.UploadUserPhotoRequest) (*userpb.UploadUserPhotoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Validate input
	if len(req.GetPhotoData()) == 0 {
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Photo data is required",
			Error:   "empty photo data",
		}, status.Error(codes.InvalidArgument, "Photo data is required")
	}

	if req.GetFileName() == "" {
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Filename is required",
			Error:   "missing filename",
		}, status.Error(codes.InvalidArgument, "Filename is required")
	}

	if req.GetDisplayOrder() < 1 || req.GetDisplayOrder() > constants.MaxPhotoDisplayOrder {
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Display order must be between 1 and 3",
			Error:   "invalid display order",
		}, status.Error(codes.InvalidArgument, "Display order must be between 1 and 3")
	}

	// Validate file
	if err := h.validateImageFile(req.GetPhotoData(), req.GetFileName()); err != nil {
		h.logger.Error("File validation failed", "error", err, "userID", userID)
		var errMsg string
		switch {
		case errors.Is(err, userErrors.ErrInvalidFileType):
			errMsg = "Unsupported file type. Allowed types: jpg, jpeg, png"
		case errors.Is(err, userErrors.ErrFileTooLarge):
			errMsg = "File size exceeds the maximum allowed size"
		default:
			errMsg = "Invalid file"
		}
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: errMsg,
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, errMsg)
	}

	// Create temporary file for the photo service
	tempFile, err := h.createTempFile(req.GetPhotoData(), req.GetFileName())
	if err != nil {
		h.logger.Error("Failed to create temp file", "error", err, "userID", userID)
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Failed to process photo",
			Error:   "temp file creation failed",
		}, status.Error(codes.Internal, "Failed to process photo")
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Create file header
	fileHeader := &multipart.FileHeader{
		Filename: req.GetFileName(),
		Size:     int64(len(req.GetPhotoData())),
		Header:   make(map[string][]string),
	}
	if req.GetContentType() != "" {
		fileHeader.Header.Set("Content-Type", req.GetContentType())
	}

	// Reset file pointer
	tempFile.Seek(0, 0)

	// Upload photo
	photoURL, err := h.photoService.UploadUserPhoto(ctx, userID, fileHeader, tempFile, int(req.GetDisplayOrder()))
	if err != nil {
		h.logger.Error("Failed to upload user photo", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrPhotoUploadFailed):
			errMsg = "Failed to upload photo"
			statusCode = codes.Internal
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.UploadUserPhotoResponse{
			Success: false,
			Message: "Failed to upload photo",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UploadUserPhotoResponse{
		Success:  true,
		Message:  "Photo uploaded successfully",
		PhotoUrl: photoURL,
	}, nil
}

// GetUserPhotos retrieves all photos for a user
func (h *PhotoHandler) GetUserPhotos(ctx context.Context, req *userpb.GetUserPhotosRequest) (*userpb.GetUserPhotosResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetUserPhotosResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Get photos
	photos, err := h.photoService.GetUserPhotos(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user photos", "error", err, "userID", userID)
		return &userpb.GetUserPhotosResponse{
			Success: false,
			Message: "Failed to retrieve photos",
			Error:   "internal server error",
		}, status.Error(codes.Internal, "Failed to retrieve photos")
	}

	// Convert to proto format
	protoPhotos := make([]*userpb.UserPhotoData, len(photos))
	for i, photo := range photos {
		protoPhotos[i] = &userpb.UserPhotoData{
			PhotoUrl:     photo.PhotoURL,
			DisplayOrder: int32(photo.DisplayOrder),
			CreatedAt:    timestamppb.New(photo.CreatedAt),
		}
	}

	return &userpb.GetUserPhotosResponse{
		Success: true,
		Message: "Photos retrieved successfully",
		Photos:  protoPhotos,
	}, nil
}

// DeleteUserPhoto deletes a specific user photo
func (h *PhotoHandler) DeleteUserPhoto(ctx context.Context, req *userpb.DeleteUserPhotoRequest) (*userpb.DeleteUserPhotoResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.DeleteUserPhotoResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Validate display order
	if req.GetDisplayOrder() < 1 || req.GetDisplayOrder() > constants.MaxPhotoDisplayOrder {
		return &userpb.DeleteUserPhotoResponse{
			Success: false,
			Message: "Display order must be between 1 and 3",
			Error:   "invalid display order",
		}, status.Error(codes.InvalidArgument, "Display order must be between 1 and 3")
	}

	// Delete photo
	err = h.photoService.DeleteUserPhoto(ctx, userID, int(req.GetDisplayOrder()))
	if err != nil {
		h.logger.Error("Failed to delete user photo", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrPhotoNotFound):
			errMsg = "Photo not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrPhotoDeleteFailed):
			errMsg = "Failed to delete photo"
			statusCode = codes.Internal
		default:
			errMsg = "Failed to delete photo"
			statusCode = codes.Internal
		}
		return &userpb.DeleteUserPhotoResponse{
			Success: false,
			Message: errMsg,
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.DeleteUserPhotoResponse{
		Success: true,
		Message: "Photo deleted successfully",
	}, nil
}
