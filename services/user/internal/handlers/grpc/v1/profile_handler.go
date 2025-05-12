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
	"google.golang.org/protobuf/types/known/timestamppb"

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

func (h *ProfileHandler) DeleteProfilePhoto(ctx context.Context, req *userpb.DeleteProfilePhotoRequest) (*userpb.DeleteProfilePhotoResponse, error) {
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
		case errors.Is(err, services.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrPhotoDeleteFailed):
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

func (h *ProfileHandler) PatchProfile(ctx context.Context, req *userpb.PatchProfileRequest) (*userpb.UpdateProfileResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdateProfileResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Convert proto wrapper types to Go pointers
	var isBride *bool
	var fullName *string
	var dateOfBirth *time.Time
	var heightCM *int
	var physicallyChallenged *bool
	var community *string
	var maritalStatus *string
	var profession *string
	var professionType *string
	var educationLevel *string
	var homeDistrict *string

	if req.IsBride != nil {
		value := req.IsBride.Value
		isBride = &value
	}

	if req.FullName != nil {
		value := req.FullName.Value
		fullName = &value
	}

	if req.DateOfBirth != nil && !req.ClearDateOfBirth {
		dob := req.DateOfBirth.AsTime()
		dateOfBirth = &dob
	}

	if req.HeightCm != nil && !req.ClearHeightCm {
		height := int(req.HeightCm.Value)
		heightCM = &height
	}

	if req.PhysicallyChallenged != nil {
		value := req.PhysicallyChallenged.Value
		physicallyChallenged = &value
	}

	if req.Community != nil {
		value := req.Community.Value
		community = &value
	}

	if req.MaritalStatus != nil {
		value := req.MaritalStatus.Value
		maritalStatus = &value
	}

	if req.Profession != nil {
		value := req.Profession.Value
		profession = &value
	}

	if req.ProfessionType != nil {
		value := req.ProfessionType.Value
		professionType = &value
	}

	if req.HighestEducationLevel != nil {
		value := req.HighestEducationLevel.Value
		educationLevel = &value
	}

	if req.HomeDistrict != nil {
		value := req.HomeDistrict.Value
		homeDistrict = &value
	}

	err = h.profileService.PatchProfile(
		ctx,
		userID,
		isBride,
		fullName,
		dateOfBirth,
		heightCM,
		physicallyChallenged,
		community,
		maritalStatus,
		profession,
		professionType,
		educationLevel,
		homeDistrict,
		req.ClearDateOfBirth,
		req.ClearHeightCm,
	)

	if err != nil {
		h.logger.Error("Failed to patch profile", "error", err, "userID", userID)
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
			Message: "Failed to patch profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UpdateProfileResponse{
		Success: true,
		Message: "Profile patched successfully",
	}, nil
}

func (h *ProfileHandler) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetProfileResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	profile, err := h.profileService.GetProfile(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get profile", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, services.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}

		return &userpb.GetProfileResponse{
			Success: false,
			Message: "Failed to get profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Convert profile model to response
	profileData := &userpb.ProfileData{
		Id:                    uint64(profile.ID),
		IsBride:               profile.IsBride,
		FullName:              profile.FullName,
		Phone:                 profile.Phone,
		PhysicallyChallenged:  profile.PhysicallyChallenged,
		Community:             string(profile.Community),
		MaritalStatus:         string(profile.MaritalStatus),
		Profession:            string(profile.Profession),
		ProfessionType:        string(profile.ProfessionType),
		HighestEducationLevel: string(profile.HighestEducationLevel),
		HomeDistrict:          string(profile.HomeDistrict),
		CreatedAt:             timestamppb.New(profile.CreatedAt),
		LastLogin:             timestamppb.New(profile.LastLogin),
	}

	// Add optional fields if present
	if profile.DateOfBirth != nil {
		profileData.DateOfBirth = timestamppb.New(*profile.DateOfBirth)
	}

	if profile.HeightCM != nil {
		profileData.HeightCm = int32(*profile.HeightCM)
	}

	if profile.ProfilePictureURL != nil {
		profileData.ProfilePictureUrl = *profile.ProfilePictureURL
	}

	return &userpb.GetProfileResponse{
		Success: true,
		Message: "Profile retrieved successfully",
		Profile: profileData,
	}, nil
}

// UpdatePartnerPreferences implements the UpdatePartnerPreferences RPC
func (h *ProfileHandler) UpdatePartnerPreferences(ctx context.Context, req *userpb.UpdatePartnerPreferencesRequest) (*userpb.UpdatePartnerPreferencesResponse, error) {
	// Extract user ID from context
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdatePartnerPreferencesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Convert request fields
	var minAgeYears, maxAgeYears, minHeightCM, maxHeightCM *int

	if req.MinAgeYears > 0 {
		age := int(req.MinAgeYears)
		minAgeYears = &age
	}
	if req.MaxAgeYears > 0 {
		age := int(req.MaxAgeYears)
		maxAgeYears = &age
	}
	if req.MinHeightCm > 0 {
		height := int(req.MinHeightCm)
		minHeightCM = &height
	}
	if req.MaxHeightCm > 0 {
		height := int(req.MaxHeightCm)
		maxHeightCM = &height
	}

	// Call service method
	err = h.profileService.UpdatePartnerPreferences(
		ctx,
		userID,
		minAgeYears,
		maxAgeYears,
		minHeightCM,
		maxHeightCM,
		req.AcceptPhysicallyChallenged,
		req.PreferredCommunities,
		req.PreferredMaritalStatus,
		req.PreferredProfessions,
		req.PreferredProfessionTypes,
		req.PreferredEducationLevels,
		req.PreferredHomeDistricts,
	)

	if err != nil {
		h.logger.Error("Failed to update partner preferences", "error", err, "userID", userID)

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

		return &userpb.UpdatePartnerPreferencesResponse{
			Success: false,
			Message: "Failed to update partner preferences",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UpdatePartnerPreferencesResponse{
		Success: true,
		Message: "Partner preferences updated successfully",
	}, nil
}

func (h *ProfileHandler) PatchPartnerPreferences(ctx context.Context, req *userpb.PatchPartnerPreferencesRequest) (*userpb.UpdatePartnerPreferencesResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdatePartnerPreferencesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	var minAgeYears, maxAgeYears, minHeightCM, maxHeightCM *int
	var acceptPhysicallyChallenged *bool

	if req.MinAgeYears != nil {
		age := int(req.MinAgeYears.Value)
		minAgeYears = &age
	}
	if req.MaxAgeYears != nil {
		age := int(req.MaxAgeYears.Value)
		maxAgeYears = &age
	}
	if req.MinHeightCm != nil {
		height := int(req.MinHeightCm.Value)
		minHeightCM = &height
	}
	if req.MaxHeightCm != nil {
		height := int(req.MaxHeightCm.Value)
		maxHeightCM = &height
	}
	if req.AcceptPhysicallyChallenged != nil {
		value := req.AcceptPhysicallyChallenged.Value
		acceptPhysicallyChallenged = &value
	}

	err = h.profileService.PatchPartnerPreferences(
		ctx,
		userID,
		minAgeYears,
		maxAgeYears,
		minHeightCM,
		maxHeightCM,
		acceptPhysicallyChallenged,
		req.PreferredCommunities,
		req.PreferredMaritalStatus,
		req.PreferredProfessions,
		req.PreferredProfessionTypes,
		req.PreferredEducationLevels,
		req.PreferredHomeDistricts,
		req.ClearPreferredCommunities,
		req.ClearPreferredMaritalStatus,
		req.ClearPreferredProfessions,
		req.ClearPreferredProfessionTypes,
		req.ClearPreferredEducationLevels,
		req.ClearPreferredHomeDistricts,
	)

	if err != nil {
		h.logger.Error("Failed to patch partner preferences", "error", err, "userID", userID)
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
		return &userpb.UpdatePartnerPreferencesResponse{
			Success: false,
			Message: "Failed to patch partner preferences",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.UpdatePartnerPreferencesResponse{
		Success: true,
		Message: "Partner preferences patched successfully",
	}, nil
}

func (h *ProfileHandler) GetPartnerPreferences(ctx context.Context, req *userpb.GetPartnerPreferencesRequest) (*userpb.GetPartnerPreferencesResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetPartnerPreferencesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	prefs, err := h.profileService.GetPartnerPreferences(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get partner preferences", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.GetPartnerPreferencesResponse{
			Success: false,
			Message: "Failed to get partner preferences",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Convert to response data
	var minAge, maxAge, minHeight, maxHeight int32
	if prefs == nil {
		// Return empty preferences if not set
		return &userpb.GetPartnerPreferencesResponse{
			Success: true,
			Message: "Partner preferences retrieved successfully",
			Preferences: &userpb.PartnerPreferencesData{
				AcceptPhysicallyChallenged: true, // Default value
				PreferredCommunities:       []string{},
				PreferredMaritalStatus:     []string{},
				PreferredProfessions:       []string{},
				PreferredProfessionTypes:   []string{},
				PreferredEducationLevels:   []string{},
				PreferredHomeDistricts:     []string{},
			},
		}, nil
	}

	if prefs.MinAgeYears != nil {
		minAge = int32(*prefs.MinAgeYears)
	}
	if prefs.MaxAgeYears != nil {
		maxAge = int32(*prefs.MaxAgeYears)
	}
	if prefs.MinHeightCM != nil {
		minHeight = int32(*prefs.MinHeightCM)
	}
	if prefs.MaxHeightCM != nil {
		maxHeight = int32(*prefs.MaxHeightCM)
	}

	// Convert enum types to strings
	communities := make([]string, len(prefs.PreferredCommunities))
	for i, c := range prefs.PreferredCommunities {
		communities[i] = string(c)
	}

	maritalStatus := make([]string, len(prefs.PreferredMaritalStatus))
	for i, s := range prefs.PreferredMaritalStatus {
		maritalStatus[i] = string(s)
	}

	professions := make([]string, len(prefs.PreferredProfessions))
	for i, p := range prefs.PreferredProfessions {
		professions[i] = string(p)
	}

	professionTypes := make([]string, len(prefs.PreferredProfessionTypes))
	for i, pt := range prefs.PreferredProfessionTypes {
		professionTypes[i] = string(pt)
	}

	educationLevels := make([]string, len(prefs.PreferredEducationLevels))
	for i, el := range prefs.PreferredEducationLevels {
		educationLevels[i] = string(el)
	}

	homeDistricts := make([]string, len(prefs.PreferredHomeDistricts))
	for i, hd := range prefs.PreferredHomeDistricts {
		homeDistricts[i] = string(hd)
	}

	return &userpb.GetPartnerPreferencesResponse{
		Success: true,
		Message: "Partner preferences retrieved successfully",
		Preferences: &userpb.PartnerPreferencesData{
			MinAgeYears:                minAge,
			MaxAgeYears:                maxAge,
			MinHeightCm:                minHeight,
			MaxHeightCm:                maxHeight,
			AcceptPhysicallyChallenged: prefs.AcceptPhysicallyChallenged,
			PreferredCommunities:       communities,
			PreferredMaritalStatus:     maritalStatus,
			PreferredProfessions:       professions,
			PreferredProfessionTypes:   professionTypes,
			PreferredEducationLevels:   educationLevels,
			PreferredHomeDistricts:     homeDistricts,
		},
	}, nil
}
