package v1

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
	userErrors "github.com/mohamedfawas/qubool-kallyanam/services/user/internal/errors"
)

type PartnerPreferencesHandler struct {
	partnerPreferencesService *services.PartnerPreferencesService
	jwtManager                *jwt.Manager
	logger                    logging.Logger
}

func NewPartnerPreferencesHandler(
	partnerPreferencesService *services.PartnerPreferencesService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *PartnerPreferencesHandler {
	return &PartnerPreferencesHandler{
		partnerPreferencesService: partnerPreferencesService,
		jwtManager:                jwtManager,
		logger:                    logger,
	}
}

// extractUserID is a helper method to extract user ID from incoming context metadata
func (h *PartnerPreferencesHandler) extractUserID(ctx context.Context) (string, error) {
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

// UpdatePartnerPreferences implements the UpdatePartnerPreferences RPC
func (h *PartnerPreferencesHandler) UpdatePartnerPreferences(ctx context.Context, req *userpb.UpdatePartnerPreferencesRequest) (*userpb.UpdatePartnerPreferencesResponse, error) {
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
	err = h.partnerPreferencesService.UpdatePartnerPreferences(
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
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrValidation):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidAgeRange):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidHeightRange):
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

// PatchPartnerPreferences implements partial update of partner preferences
func (h *PartnerPreferencesHandler) PatchPartnerPreferences(ctx context.Context, req *userpb.PatchPartnerPreferencesRequest) (*userpb.UpdatePartnerPreferencesResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdatePartnerPreferencesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Convert optional fields to pointers
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

	// Call service method
	err = h.partnerPreferencesService.PatchPartnerPreferences(
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
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrValidation):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidAgeRange):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidHeightRange):
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

// GetPartnerPreferences retrieves current partner preferences
func (h *PartnerPreferencesHandler) GetPartnerPreferences(ctx context.Context, req *userpb.GetPartnerPreferencesRequest) (*userpb.GetPartnerPreferencesResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetPartnerPreferencesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	prefs, err := h.partnerPreferencesService.GetPartnerPreferences(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get partner preferences", "error", err, "userID", userID)
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
