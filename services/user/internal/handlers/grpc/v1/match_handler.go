package v1

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
)

func (h *ProfileHandler) GetRecommendedMatches(ctx context.Context, req *userpb.GetRecommendedMatchesRequest) (*userpb.GetRecommendedMatchesResponse, error) {
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

	// Call service layer
	profiles, pagination, err := h.matchmakingService.GetRecommendedMatches(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get recommended matches", "error", err, "userID", userID)
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

func (h *ProfileHandler) RecordMatchAction(ctx context.Context, req *userpb.RecordMatchActionRequest) (*userpb.RecordMatchActionResponse, error) {
	userID, err := h.extractUserID(ctx)
	if err != nil {
		h.logger.Error("Authentication failed", "error", err)
		return &userpb.RecordMatchActionResponse{
			Success: false,
			Message: "Authentication required",
			Error:   err.Error(),
		}, err
	}

	profileID := uint(req.GetProfileId()) // Changed from GetTargetUserId()
	action := req.GetAction()

	isMutualMatch, err := h.matchmakingService.RecordMatchAction(ctx, userID, profileID, action)
	if err != nil {
		h.logger.Error("Failed to record match action", "error", err, "userID", userID, "profileID", profileID, "action", action)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrInvalidActionType):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrMatchActionFailed):
			errMsg = "Failed to record match action"
			statusCode = codes.Internal
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

func (h *ProfileHandler) GetMatchHistory(ctx context.Context, req *userpb.GetMatchHistoryRequest) (*userpb.GetMatchHistoryResponse, error) {
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
	statusFilter := req.GetStatus() // Changed from 'status' to 'statusFilter'

	items, pagination, err := h.matchmakingService.GetMatchHistory(ctx, userID, statusFilter, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get match history", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		default:
			errMsg = "Internal server error"
			statusCode = codes.Internal
		}
		return &userpb.GetMatchHistoryResponse{
			Success: false,
			Message: "Failed to get match history",
			Error:   errMsg,
		}, status.Error(statusCode, errMsg) // Now 'status' refers to the gRPC status package
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

func (h *ProfileHandler) UpdateMatchAction(ctx context.Context, req *userpb.UpdateMatchActionRequest) (*userpb.UpdateMatchActionResponse, error) {
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

	isMutualMatch, wasMutualMatchBroken, err := h.matchmakingService.UpdateMatchAction(ctx, userID, profileID, action)
	if err != nil {
		h.logger.Error("Failed to update match action", "error", err, "userID", userID, "profileID", profileID, "action", action)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrInvalidActionType):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
		case errors.Is(err, services.ErrMatchActionFailed):
			errMsg = "Failed to update match action"
			statusCode = codes.Internal
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

func (h *ProfileHandler) GetMutualMatches(ctx context.Context, req *userpb.GetMutualMatchesRequest) (*userpb.GetMutualMatchesResponse, error) {
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

	matches, pagination, err := h.matchmakingService.GetMutualMatches(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get mutual matches", "error", err, "userID", userID)
		var errMsg string
		var statusCode codes.Code
		switch {
		case errors.Is(err, services.ErrInvalidInput):
			errMsg = err.Error()
			statusCode = codes.InvalidArgument
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
