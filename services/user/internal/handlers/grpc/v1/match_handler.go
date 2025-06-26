package v1

import (
	"context"
	"errors"
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

type MatchHandler struct {
	matchmakingService *services.MatchmakingService
	jwtManager         *jwt.Manager
	logger             logging.Logger
}

func NewMatchHandler(
	matchmakingService *services.MatchmakingService,
	jwtManager *jwt.Manager,
	logger logging.Logger,
) *MatchHandler {
	return &MatchHandler{
		matchmakingService: matchmakingService,
		jwtManager:         jwtManager,
		logger:             logger,
	}
}

// extractUserID is a helper method to extract user ID from incoming context metadata
func (h *MatchHandler) extractUserID(ctx context.Context) (string, error) {
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

// validatePaginationParams validates limit and offset parameters
func (h *MatchHandler) validatePaginationParams(limit, offset int32) error {
	if limit <= 0 || limit > constants.MaxPaginationLimit {
		return userErrors.ErrInvalidPaginationLimit
	}
	if offset < 0 {
		return userErrors.ErrInvalidPaginationOffset
	}
	return nil
}

// GetRecommendedMatches retrieves recommended matches for the user
func (h *MatchHandler) GetRecommendedMatches(ctx context.Context, req *userpb.GetRecommendedMatchesRequest) (*userpb.GetRecommendedMatchesResponse, error) {
	// Authentication
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetRecommendedMatchesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	// Extract and validate request parameters
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())

	// Set default limit if not provided
	if limit == 0 {
		limit = constants.DefaultPaginationLimit
	}

	// Validate pagination parameters
	if err := h.validatePaginationParams(int32(limit), int32(offset)); err != nil {
		h.logger.Error("Invalid pagination parameters", "error", err, "userID", userID, "limit", limit, "offset", offset)
		return &userpb.GetRecommendedMatchesResponse{
			Success: false,
			Message: "Invalid pagination parameters",
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service layer
	profiles, pagination, err := h.matchmakingService.GetRecommendedMatches(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get recommended matches", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrNoMatches):
			errMsg = "No matches found"
			statusCode = codes.NotFound
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.GetRecommendedMatchesResponse{
			Success: false,
			Message: "Failed to get recommended matches",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// If no profiles were found, return a success response with empty list
	if len(profiles) == 0 {
		return &userpb.GetRecommendedMatchesResponse{
			Success:  true,
			Message:  "No matches found based on compatibility scoring",
			Profiles: []*userpb.RecommendedProfileData{},
			Pagination: &userpb.PaginationData{
				Total:   0,
				Limit:   int32(limit),
				Offset:  int32(offset),
				HasMore: false,
			},
		}, nil
	}

	// Transform domain models to protobuf response
	profileResponses := make([]*userpb.RecommendedProfileData, len(profiles))
	for i, profile := range profiles {
		var heightCM int32
		if profile.HeightCM != nil {
			heightCM = int32(*profile.HeightCM)
		}

		var profilePictureURL string
		if profile.ProfilePictureURL != nil {
			profilePictureURL = *profile.ProfilePictureURL
		}

		profileResponses[i] = &userpb.RecommendedProfileData{
			ProfileId:             uint64(profile.ID),
			FullName:              profile.FullName,
			Age:                   int32(profile.Age),
			HeightCm:              heightCM,
			PhysicallyChallenged:  profile.PhysicallyChallenged,
			Community:             string(profile.Community),
			MaritalStatus:         string(profile.MaritalStatus),
			Profession:            string(profile.Profession),
			ProfessionType:        string(profile.ProfessionType),
			HighestEducationLevel: string(profile.HighestEducationLevel),
			HomeDistrict:          string(profile.HomeDistrict),
			ProfilePictureUrl:     profilePictureURL,
			LastLogin:             timestamppb.New(profile.LastLogin),
			MatchReasons:          profile.MatchReasons,
		}
	}

	// Create pagination information
	paginationResponse := &userpb.PaginationData{
		Total:   int32(pagination.Total),
		Limit:   int32(pagination.Limit),
		Offset:  int32(pagination.Offset),
		HasMore: pagination.HasMore,
	}

	// Return successful response
	return &userpb.GetRecommendedMatchesResponse{
		Success:    true,
		Message:    "Recommended matches retrieved successfully",
		Profiles:   profileResponses,
		Pagination: paginationResponse,
	}, nil
}

// RecordMatchAction records a user's action on a profile (like/pass)
func (h *MatchHandler) RecordMatchAction(ctx context.Context, req *userpb.RecordMatchActionRequest) (*userpb.RecordMatchActionResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	profileID := uint(req.GetProfileId())
	action := req.GetAction()

	// Validate input
	if profileID == 0 {
		h.logger.Error("Invalid profile ID", "userID", userID, "profileID", profileID)
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Invalid profile ID",
			Error:   "Profile ID must be greater than 0",
		}, status.Error(codes.InvalidArgument, "Profile ID must be greater than 0")
	}

	if action == "" {
		h.logger.Error("Invalid action", "userID", userID, "action", action)
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Invalid action",
			Error:   "Action is required",
		}, status.Error(codes.InvalidArgument, "Action is required")
	}

	// Validate action type
	validAction := false
	for _, validActionType := range constants.ValidMatchActions {
		if action == validActionType {
			validAction = true
			break
		}
	}

	if !validAction {
		h.logger.Error("Invalid action type", "userID", userID, "action", action)
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Invalid action type",
			Error:   "Action must be 'like' or 'pass'",
		}, status.Error(codes.InvalidArgument, "Action must be 'like' or 'pass'")
	}

	isMutualMatch, err := h.matchmakingService.RecordMatchAction(ctx, userID, profileID, action)
	if err != nil {
		h.logger.Error("Failed to record match action", "error", err, "userID", userID, "profileID", profileID, "action", action)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidActionType):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrMatchActionFailed):
			errMsg = "Failed to record match action"
			statusCode = codes.Internal
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Target profile not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Failed to record match action",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	message := "Match action recorded successfully"
	if isMutualMatch {
		message = "It's a match! Both of you liked each other"
	}

	return &userpb.RecordMatchActionResponse{
		Success:       true,
		Message:       message,
		IsMutualMatch: isMutualMatch,
	}, nil
}

// GetMatchHistory retrieves the user's match history
func (h *MatchHandler) GetMatchHistory(ctx context.Context, req *userpb.GetMatchHistoryRequest) (*userpb.GetMatchHistoryResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetMatchHistoryResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	limit := int(req.GetLimit())
	offset := int(req.GetOffset())
	statusFilter := req.GetStatus()

	// Set default limit if not provided
	if limit == 0 {
		limit = constants.DefaultPaginationLimit
	}

	// Validate pagination parameters
	if err := h.validatePaginationParams(int32(limit), int32(offset)); err != nil {
		h.logger.Error("Invalid pagination parameters", "error", err, "userID", userID, "limit", limit, "offset", offset)
		return &userpb.GetMatchHistoryResponse{
			Success: false,
			Message: "Invalid pagination parameters",
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate status filter if provided
	if statusFilter != "" {
		validStatus := false
		for _, validStatusType := range constants.ValidMatchActions {
			if statusFilter == validStatusType {
				validStatus = true
				break
			}
		}
		if !validStatus {
			h.logger.Error("Invalid status filter", "userID", userID, "status", statusFilter)
			return &userpb.GetMatchHistoryResponse{
				Success: false,
				Message: "Invalid status filter",
				Error:   "Status must be 'like' or 'pass'",
			}, status.Error(codes.InvalidArgument, "Status must be 'like' or 'pass'")
		}
	}

	items, pagination, err := h.matchmakingService.GetMatchHistory(ctx, userID, statusFilter, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get match history", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.GetMatchHistoryResponse{
			Success: false,
			Message: "Failed to get match history",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Transform to protobuf response
	matchItems := make([]*userpb.MatchHistoryItem, len(items))
	for i, item := range items {
		var heightCM int32
		if item.HeightCM != nil {
			heightCM = int32(*item.HeightCM)
		}

		var profilePictureURL string
		if item.ProfilePictureURL != nil {
			profilePictureURL = *item.ProfilePictureURL
		}

		matchItems[i] = &userpb.MatchHistoryItem{
			ProfileId:             uint64(item.ProfileID),
			FullName:              item.FullName,
			Age:                   int32(item.Age),
			HeightCm:              heightCM,
			PhysicallyChallenged:  item.PhysicallyChallenged,
			Community:             string(item.Community),
			MaritalStatus:         string(item.MaritalStatus),
			Profession:            string(item.Profession),
			ProfessionType:        string(item.ProfessionType),
			HighestEducationLevel: string(item.HighestEducationLevel),
			HomeDistrict:          string(item.HomeDistrict),
			ProfilePictureUrl:     profilePictureURL,
			Action:                string(item.Action),
			ActionDate:            timestamppb.New(item.ActionDate),
		}
	}

	paginationResponse := &userpb.PaginationData{
		Total:   int32(pagination.Total),
		Limit:   int32(pagination.Limit),
		Offset:  int32(pagination.Offset),
		HasMore: pagination.HasMore,
	}

	return &userpb.GetMatchHistoryResponse{
		Success:    true,
		Message:    "Match history retrieved successfully",
		Matches:    matchItems,
		Pagination: paginationResponse,
	}, nil
}

// UpdateMatchAction updates a user's previous action on a profile
func (h *MatchHandler) UpdateMatchAction(ctx context.Context, req *userpb.UpdateMatchActionRequest) (*userpb.UpdateMatchActionResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.UpdateMatchActionResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	profileID := uint(req.GetProfileId())
	action := req.GetAction()

	// Validate input
	if profileID == 0 {
		h.logger.Error("Invalid profile ID", "userID", userID, "profileID", profileID)
		return &userpb.UpdateMatchActionResponse{
			Success: false,
			Message: "Invalid profile ID",
			Error:   "Profile ID must be greater than 0",
		}, status.Error(codes.InvalidArgument, "Profile ID must be greater than 0")
	}

	if action == "" {
		h.logger.Error("Invalid action", "userID", userID, "action", action)
		return &userpb.UpdateMatchActionResponse{
			Success: false,
			Message: "Invalid action",
			Error:   "Action is required",
		}, status.Error(codes.InvalidArgument, "Action is required")
	}

	// Validate action type
	validAction := false
	for _, validActionType := range constants.ValidMatchActions {
		if action == validActionType {
			validAction = true
			break
		}
	}

	if !validAction {
		h.logger.Error("Invalid action type", "userID", userID, "action", action)
		return &userpb.UpdateMatchActionResponse{
			Success: false,
			Message: "Invalid action type",
			Error:   "Action must be 'like' or 'pass'",
		}, status.Error(codes.InvalidArgument, "Action must be 'like' or 'pass'")
	}

	isMutualMatch, wasMutualMatchBroken, err := h.matchmakingService.UpdateMatchAction(ctx, userID, profileID, action)
	if err != nil {
		h.logger.Error("Failed to update match action", "error", err, "userID", userID, "profileID", profileID, "action", action)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrInvalidActionType):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrMatchActionFailed):
			errMsg = "Failed to update match action"
			statusCode = codes.Internal
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Target profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrMatchNotFound):
			errMsg = "Previous match action not found"
			statusCode = codes.NotFound
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.UpdateMatchActionResponse{
			Success: false,
			Message: "Failed to update match action",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	var message string
	if wasMutualMatchBroken {
		message = "Match action updated successfully. Previous mutual match has been removed."
	} else if isMutualMatch {
		message = "Match action updated successfully. It's a match! Both of you liked each other."
	} else {
		message = "Match action updated successfully"
	}

	return &userpb.UpdateMatchActionResponse{
		Success:              true,
		Message:              message,
		IsMutualMatch:        isMutualMatch,
		WasMutualMatchBroken: wasMutualMatchBroken,
	}, nil
}

// GetMutualMatches retrieves all mutual matches for the user
func (h *MatchHandler) GetMutualMatches(ctx context.Context, req *userpb.GetMutualMatchesRequest) (*userpb.GetMutualMatchesResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.GetMutualMatchesResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	limit := int(req.GetLimit())
	offset := int(req.GetOffset())

	// Set default limit if not provided
	if limit == 0 {
		limit = constants.DefaultPaginationLimit
	}

	// Validate pagination parameters
	if err := h.validatePaginationParams(int32(limit), int32(offset)); err != nil {
		h.logger.Error("Invalid pagination parameters", "error", err, "userID", userID, "limit", limit, "offset", offset)
		return &userpb.GetMutualMatchesResponse{
			Success: false,
			Message: "Invalid pagination parameters",
			Error:   err.Error(),
		}, status.Error(codes.InvalidArgument, err.Error())
	}

	matches, pagination, err := h.matchmakingService.GetMutualMatches(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get mutual matches", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, userErrors.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, userErrors.ErrProfileNotFound):
			errMsg = "Profile not found"
			statusCode = codes.NotFound
		case errors.Is(err, userErrors.ErrNoMatches):
			errMsg = "No mutual matches found"
			statusCode = codes.NotFound
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.GetMutualMatchesResponse{
			Success: false,
			Message: "Failed to get mutual matches",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg)
	}

	// Convert to protobuf response
	matchResponses := make([]*userpb.MutualMatchData, len(matches))
	for i, match := range matches {
		var heightCM int32
		if match.HeightCM != nil {
			heightCM = int32(*match.HeightCM)
		}

		var profilePictureURL string
		if match.ProfilePictureURL != nil {
			profilePictureURL = *match.ProfilePictureURL
		}

		matchResponses[i] = &userpb.MutualMatchData{
			ProfileId:             uint64(match.ProfileID),
			FullName:              match.FullName,
			Age:                   int32(match.Age),
			HeightCm:              heightCM,
			PhysicallyChallenged:  match.PhysicallyChallenged,
			Community:             string(match.Community),
			MaritalStatus:         string(match.MaritalStatus),
			Profession:            string(match.Profession),
			ProfessionType:        string(match.ProfessionType),
			HighestEducationLevel: string(match.HighestEducationLevel),
			HomeDistrict:          string(match.HomeDistrict),
			ProfilePictureUrl:     profilePictureURL,
			LastLogin:             timestamppb.New(match.LastLogin),
			MatchedAt:             timestamppb.New(match.MatchedAt),
		}
	}

	paginationResponse := &userpb.PaginationData{
		Total:   int32(pagination.Total),
		Limit:   int32(pagination.Limit),
		Offset:  int32(pagination.Offset),
		HasMore: pagination.HasMore,
	}

	return &userpb.GetMutualMatchesResponse{
		Success:    true,
		Message:    "Mutual matches retrieved successfully",
		Matches:    matchResponses,
		Pagination: paginationResponse,
	}, nil
}
