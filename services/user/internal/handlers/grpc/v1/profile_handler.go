// user/internal/handlers/grpc/v1/profile_handler.go
package v1

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
)

type ProfileHandler struct {
	userpb.UnimplementedUserServiceServer
	profileService *services.ProfileService
	photoService   *services.PhotoService
	jwtManager     *jwt.Manager
	logger         logging.Logger
}

func NewProfileHandler(
	profileService *services.ProfileService,
	photoService *services.PhotoService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		photoService:   photoService,
		jwtManager:     jwtManager,
		logger:         logger,
	}
}

// extractUserID is a helper method to extract user ID from incoming context metadata
func (h *ProfileHandler) extractUserID(ctx context.Context) (string, error) {
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
	// This keeps backward compatibility during transition
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

func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	// Extract user ID from authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdateProfileResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	var dateOfBirth *time.Time
	if req.GetDateOfBirth() != nil {
		dob := req.GetDateOfBirth().AsTime()
		dateOfBirth = &dob
	}

	var heightCM *int
	if req.GetHeightCm() > 0 {
		height := int(req.GetHeightCm())
		heightCM = &height
	}

	err = h.profileService.UpdateProfile(
		ctx,
		userID,
		req.GetIsBride(),
		req.GetFullName(),
		dateOfBirth,
		heightCM,
		req.GetPhysicallyChallenged(),
		req.GetCommunity(),
		req.GetMaritalStatus(),
		req.GetProfession(),
		req.GetProfessionType(),
		req.GetHighestEducationLevel(),
		req.GetHomeDistrict(),
	)

	if err != nil {
		h.logger.Error("Failed to update profile", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrValidation):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.UpdateProfileResponse{
			Success: false,
			Message: "Failed to update profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UpdateProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
	}, nil
}

// UploadProfilePhoto handles the profile photo upload
func (h *ProfileHandler) UploadProfilePhoto(ctx context.Context, req *userpb.UploadProfilePhotoRequest) (*userpb.UploadProfilePhotoResponse, error) {
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

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(req.GetFileName()))
	validExt := false
	for _, allowedExt := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		h.logger.Error("Invalid file type", "extension", ext, "userID", userID)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "Unsupported file type. Allowed types: jpg, jpeg, png, gif, webp",
			Error:   "invalid file type",
		}, status.Error(codes.InvalidArgument, "Unsupported file type")
	}

	// Convert binary data to multipart file
	fileData := req.GetPhotoData()

	// Check file size (5MB max)
	const maxFileSize = 5 * 1024 * 1024
	if len(fileData) > maxFileSize {
		h.logger.Error("File too large", "size", len(fileData), "userID", userID)
		return &userpb.UploadProfilePhotoResponse{
			Success: false,
			Message: "File size exceeds the maximum allowed size of 5MB",
			Error:   "file too large",
		}, status.Error(codes.InvalidArgument, "File too large")
	}

	// For MVP: Create a temporary file
	tempFile, err := createTempFile(fileData, req.GetFileName())
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
		contentType = http.DetectContentType(fileData)
	}

	// Create a multipart file header
	header := &multipart.FileHeader{
		Filename: req.GetFileName(),
		Size:     int64(len(fileData)),
	}

	// Upload the photo
	photoURL, err := h.photoService.UploadProfilePhoto(ctx, userID, header, tempFile)
	if err != nil {
		h.logger.Error("Failed to upload profile photo", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrPhotoUploadFailed):
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

// createTempFile creates a temporary file from the given data
func createTempFile(data []byte, filename string) (*os.File, error) {
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
