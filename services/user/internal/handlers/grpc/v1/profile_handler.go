package v1

import (
	"context"
	"errors"
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
	userErrors "github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
)

type ProfileHandler struct {
	userpb.UnimplementedUserServiceServer
	profileService *services.ProfileService
	jwtManager     *jwt.Manager
	logger         logging.Logger
}

func NewProfileHandler(
	profileService *services.ProfileService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
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

// UpdateProfile updates the user's complete profile information
func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdateProfileResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Convert height from proto to pointer
	var heightCM *int
	if req.GetHeightCm() > 0 {
		height := int(req.GetHeightCm())
		heightCM = &height
	}

	// Call service layer
	err = h.profileService.UpdateProfile(
		ctx,
		userID,
		req.GetIsBride(),
		req.GetFullName(),
		req.GetDateOfBirth(),
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
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrValidation):
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

// PatchProfile partially updates the user's profile information
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

	// Handle date of birth validation
	if req.DateOfBirth != "" && !req.ClearDateOfBirth {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err != nil {
			h.logger.Error("Invalid date format", "error", err, "userID", userID, "dateOfBirth", req.DateOfBirth)
			return &userpb.UpdateProfileResponse{
				Success: false,
				Message: "Invalid date format",
				Error:   "Date must be in YYYY-MM-DD format",
			}, status.Error(codes.InvalidArgument, "Invalid date format")
		}
		req.DateOfBirth = dob.Format("2006-01-02")
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

	// Call service layer
	err = h.profileService.PatchProfile(
		ctx,
		userID,
		isBride,
		fullName,
		req.DateOfBirth,
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
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrValidation):
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

// GetProfile retrieves the authenticated user's profile
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
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
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
		profileData.DateOfBirth = profile.DateOfBirth.Format("2006-01-02")
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

// GetProfileByID resolves public profile ID to user UUID
func (h *ProfileHandler) GetProfileByID(ctx context.Context, req *userpb.GetProfileByIDRequest) (*userpb.GetProfileByIDResponse, error) {
	h.logger.Info("GetProfileByID gRPC request", "profileID", req.ProfileId)

	// Validate request
	if req.ProfileId == 0 {
		h.logger.Error("Invalid profile ID", "profileID", req.ProfileId)
		return &userpb.GetProfileByIDResponse{
			Success: false,
			Message: "Invalid profile ID",
			Error:   "Profile ID must be greater than 0",
		}, status.Error(codes.InvalidArgument, "Profile ID must be greater than 0")
	}

	// Call service
	userUUID, err := h.profileService.GetUserUUIDByProfileID(ctx, req.ProfileId)
	if err != nil {
		h.logger.Error("Failed to get user UUID by profile ID", "error", err, "profileID", req.ProfileId)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = "Invalid profile ID"
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Failed to resolve profile ID"
			statusCode = codes.Internal
		}

		return &userpb.GetProfileByIDResponse{
			Success: false,
			Message: "Failed to resolve profile ID",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	return &userpb.GetProfileByIDResponse{
		Success:  true,
		Message:  "Profile ID resolved successfully",
		UserUuid: userUUID,
	}, nil
}

// GetBasicProfile gets basic profile information by user UUID
func (h *ProfileHandler) GetBasicProfile(ctx context.Context, req *userpb.GetBasicProfileRequest) (*userpb.GetBasicProfileResponse, error) {
	h.logger.Info("GetBasicProfile gRPC request", "userUUID", req.UserUuid)

	// Validate request
	if req.UserUuid == "" {
		h.logger.Error("Invalid user UUID", "userUUID", req.UserUuid)
		return &userpb.GetBasicProfileResponse{
			Success: false,
			Message: "Invalid user UUID",
			Error:   "User UUID is required",
		}, status.Error(codes.InvalidArgument, "User UUID is required")
	}

	// Call service
	profile, err := h.profileService.GetBasicProfileByUUID(ctx, req.UserUuid)
	if err != nil {
		h.logger.Error("Failed to get basic profile", "error", err, "userUUID", req.UserUuid)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = "Invalid user UUID"
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Failed to get profile"
			statusCode = codes.Internal
		}

		return &userpb.GetBasicProfileResponse{
			Success: false,
			Message: "Failed to get basic profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build response
	basicProfile := &userpb.BasicProfileData{
		Id:       uint64(profile.ID),
		FullName: profile.FullName,
		IsActive: !profile.IsDeleted,
	}

	if profile.ProfilePictureURL != nil {
		basicProfile.ProfilePictureUrl = *profile.ProfilePictureURL
	}

	return &userpb.GetBasicProfileResponse{
		Success: true,
		Message: "Basic profile retrieved successfully",
		Profile: basicProfile,
	}, nil
}

// GetDetailedProfile gets comprehensive profile information by profile ID
func (h *ProfileHandler) GetDetailedProfile(ctx context.Context, req *userpb.GetDetailedProfileRequest) (*userpb.GetDetailedProfileResponse, error) {
	h.logger.Info("GetDetailedProfile gRPC request", "profileID", req.ProfileId)

	// Validate request
	if req.ProfileId == 0 {
		h.logger.Error("Invalid profile ID", "profileID", req.ProfileId)
		return &userpb.GetDetailedProfileResponse{
			Success: false,
			Message: "Invalid profile ID",
			Error:   "Profile ID must be greater than 0",
		}, status.Error(codes.InvalidArgument, "Profile ID must be greater than 0")
	}

	// Call service
	detailedProfile, err := h.profileService.GetDetailedProfileByID(ctx, req.ProfileId)
	if err != nil {
		h.logger.Error("Failed to get detailed profile", "error", err, "profileID", req.ProfileId)

		var errMsg string
		var statusCode codes.Code

		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = "Invalid profile ID"
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Failed to get detailed profile"
			statusCode = codes.Internal
		}

		return &userpb.GetDetailedProfileResponse{
			Success: false,
			Message: "Failed to get detailed profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build the response
	responseProfile := &userpb.DetailedProfileData{
		Id:                    uint64(detailedProfile.ID),
		IsBride:               detailedProfile.IsBride,
		FullName:              detailedProfile.FullName,
		PhysicallyChallenged:  detailedProfile.PhysicallyChallenged,
		Community:             string(detailedProfile.Community),
		MaritalStatus:         string(detailedProfile.MaritalStatus),
		Profession:            string(detailedProfile.Profession),
		ProfessionType:        string(detailedProfile.ProfessionType),
		HighestEducationLevel: string(detailedProfile.HighestEducationLevel),
		HomeDistrict:          string(detailedProfile.HomeDistrict),
		LastLogin:             timestamppb.New(detailedProfile.LastLogin),
		Age:                   int32(detailedProfile.Age),
	}

	// Add optional fields
	if detailedProfile.DateOfBirth != nil {
		responseProfile.DateOfBirth = detailedProfile.DateOfBirth.Format("2006-01-02")
	}

	if detailedProfile.HeightCM != nil {
		responseProfile.HeightCm = int32(*detailedProfile.HeightCM)
	}

	if detailedProfile.ProfilePictureURL != nil {
		responseProfile.ProfilePictureUrl = *detailedProfile.ProfilePictureURL
	}

	// Add partner preferences if present
	if detailedProfile.PartnerPreferences != nil {
		prefs := detailedProfile.PartnerPreferences
		var minAge, maxAge, minHeight, maxHeight int32

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

		responseProfile.PartnerPreferences = &userpb.PartnerPreferencesData{
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
		}
	}

	// Add additional photos if present
	if len(detailedProfile.AdditionalPhotos) > 0 {
		photos := make([]*userpb.UserPhotoData, len(detailedProfile.AdditionalPhotos))
		for i, photo := range detailedProfile.AdditionalPhotos {
			photos[i] = &userpb.UserPhotoData{
				PhotoUrl:     photo.PhotoURL,
				DisplayOrder: int32(photo.DisplayOrder),
				CreatedAt:    timestamppb.New(photo.CreatedAt),
			}
		}
		responseProfile.AdditionalPhotos = photos
	}

	// Add intro video if present
	if detailedProfile.IntroVideo != nil {
		video := detailedProfile.IntroVideo
		responseProfile.IntroVideo = &userpb.UserVideoData{
			VideoUrl:  video.VideoURL,
			FileName:  video.FileName,
			FileSize:  video.FileSize,
			CreatedAt: timestamppb.New(video.CreatedAt),
		}
		if video.DurationSeconds != nil {
			responseProfile.IntroVideo.DurationSeconds = int32(*video.DurationSeconds)
		}
	}

	return &userpb.GetDetailedProfileResponse{
		Success: true,
		Message: "Detailed profile retrieved successfully",
		Profile: responseProfile,
	}, nil
}

// GetProfileForAdmin handles admin requests for detailed user profile information
// This method bypasses normal authentication and returns complete user data
func (h *ProfileHandler) GetProfileForAdmin(ctx context.Context, req *userpb.GetProfileForAdminRequest) (*userpb.GetDetailedProfileResponse, error) {
	h.logger.Info("Admin requesting detailed user profile", "user_uuid", req.GetUserUuid())

	// Validate input
	if req.GetUserUuid() == "" {
		h.logger.Error("Missing user UUID in admin request")
		return &userpb.GetDetailedProfileResponse{
			Success: false,
			Message: "Invalid request",
			Error:   "User UUID is required",
		}, status.Error(codes.InvalidArgument, "User UUID is required")
	}

	// Call service layer (no authentication check for admin)
	detailedProfile, err := h.profileService.GetDetailedProfileByUUID(ctx, req.GetUserUuid())
	if err != nil {
		h.logger.Error("Failed to get detailed profile for admin", "error", err, "user_uuid", req.GetUserUuid())

		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = "Invalid user UUID"
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Failed to get detailed profile"
			statusCode = codes.Internal
		}

		return &userpb.GetDetailedProfileResponse{
			Success: false,
			Message: "Failed to get detailed profile",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Build the response using the same pattern as GetDetailedProfile
	responseProfile := &userpb.DetailedProfileData{
		Id:                    uint64(detailedProfile.ID),
		IsBride:               detailedProfile.IsBride,
		FullName:              detailedProfile.FullName,
		PhysicallyChallenged:  detailedProfile.PhysicallyChallenged,
		Community:             string(detailedProfile.Community),
		MaritalStatus:         string(detailedProfile.MaritalStatus),
		Profession:            string(detailedProfile.Profession),
		ProfessionType:        string(detailedProfile.ProfessionType),
		HighestEducationLevel: string(detailedProfile.HighestEducationLevel),
		HomeDistrict:          string(detailedProfile.HomeDistrict),
		LastLogin:             timestamppb.New(detailedProfile.LastLogin),
		Age:                   int32(detailedProfile.Age),
	}

	// Add optional fields
	if detailedProfile.DateOfBirth != nil {
		responseProfile.DateOfBirth = detailedProfile.DateOfBirth.Format("2006-01-02")
	}

	if detailedProfile.HeightCM != nil {
		responseProfile.HeightCm = int32(*detailedProfile.HeightCM)
	}

	if detailedProfile.ProfilePictureURL != nil {
		responseProfile.ProfilePictureUrl = *detailedProfile.ProfilePictureURL
	}

	// Add partner preferences if present
	if detailedProfile.PartnerPreferences != nil {
		prefs := detailedProfile.PartnerPreferences
		var minAge, maxAge, minHeight, maxHeight int32

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

		responseProfile.PartnerPreferences = &userpb.PartnerPreferencesData{
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
		}
	}

	// Add additional photos if present
	if len(detailedProfile.AdditionalPhotos) > 0 {
		photos := make([]*userpb.UserPhotoData, len(detailedProfile.AdditionalPhotos))
		for i, photo := range detailedProfile.AdditionalPhotos {
			photos[i] = &userpb.UserPhotoData{
				PhotoUrl:     photo.PhotoURL,
				DisplayOrder: int32(photo.DisplayOrder),
				CreatedAt:    timestamppb.New(photo.CreatedAt),
			}
		}
		responseProfile.AdditionalPhotos = photos
	}

	// Add intro video if present
	if detailedProfile.IntroVideo != nil {
		video := detailedProfile.IntroVideo
		responseProfile.IntroVideo = &userpb.UserVideoData{
			VideoUrl:  video.VideoURL,
			FileName:  video.FileName,
			FileSize:  video.FileSize,
			CreatedAt: timestamppb.New(video.CreatedAt),
		}
		if video.DurationSeconds != nil {
			responseProfile.IntroVideo.DurationSeconds = int32(*video.DurationSeconds)
		}
	}

	h.logger.Info("Successfully retrieved detailed profile for admin", "user_uuid", req.GetUserUuid())

	return &userpb.GetDetailedProfileResponse{
		Success: true,
		Message: "Detailed profile retrieved successfully",
		Profile: responseProfile,
	}, nil
}
