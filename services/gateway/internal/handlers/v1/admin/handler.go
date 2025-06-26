package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/admin/v1"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/admin"
)

// Handler handles admin-related HTTP requests
type Handler struct {
	adminClient *admin.Client
	logger      logging.Logger
}

// NewHandler creates a new admin handler
func NewHandler(adminClient *admin.Client, logger logging.Logger) *Handler {
	return &Handler{
		adminClient: adminClient,
		logger:      logger,
	}
}

// GetUsers handles GET /api/v1/admin/users
func (h *Handler) GetUsers(c *gin.Context) {
	// Parse query parameters
	var req GetUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Invalid get users request parameters", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request parameters", err))
		return
	}

	// Convert to protobuf request
	pbReq := &adminpb.GetUsersRequest{
		Limit:     req.Limit,
		Offset:    req.Offset,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Set optional boolean filters
	if req.IsActive != nil {
		pbReq.IsActive = req.IsActive
	}
	if req.VerifiedOnly != nil {
		pbReq.VerifiedOnly = req.VerifiedOnly
	}
	if req.PremiumOnly != nil {
		pbReq.PremiumOnly = req.PremiumOnly
	}

	// Set date filters
	if req.CreatedAfter != nil {
		pbReq.CreatedAfter = timestamppb.New(*req.CreatedAfter)
	}
	if req.CreatedBefore != nil {
		pbReq.CreatedBefore = timestamppb.New(*req.CreatedBefore)
	}
	if req.LastLoginAfter != nil {
		pbReq.LastLoginAfter = timestamppb.New(*req.LastLoginAfter)
	}
	if req.LastLoginBefore != nil {
		pbReq.LastLoginBefore = timestamppb.New(*req.LastLoginBefore)
	}

	// Call admin service
	resp, err := h.adminClient.GetUsers(c.Request.Context(), pbReq)
	if err != nil {
		h.logger.Error("Failed to get users", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Convert response
	response := h.convertGetUsersResponse(resp)
	pkghttp.Success(c, http.StatusOK, resp.Message, response)
}

// GetUser handles GET /api/v1/admin/users/{id}
func (h *Handler) GetUser(c *gin.Context) {
	identifier := c.Param("id")
	if identifier == "" {
		pkghttp.Error(c, pkghttp.NewBadRequest("User identifier is required", nil))
		return
	}

	// Call admin service
	resp, err := h.adminClient.GetUser(c.Request.Context(), &adminpb.GetUserRequest{
		Identifier:     identifier,
		IdentifierType: "", // Let service auto-detect
	})
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "identifier", identifier)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Convert response
	response := h.convertGetUserResponse(resp)
	pkghttp.Success(c, http.StatusOK, resp.Message, response)
}

// SearchUsers handles GET /api/v1/admin/users with search parameter
func (h *Handler) SearchUsers(c *gin.Context) {
	search := c.Query("search")
	if search == "" {
		pkghttp.Error(c, pkghttp.NewBadRequest("Search parameter is required", nil))
		return
	}

	// Call admin service with search identifier
	resp, err := h.adminClient.GetUser(c.Request.Context(), &adminpb.GetUserRequest{
		Identifier:     search,
		IdentifierType: "", // Let service auto-detect email/phone
	})
	if err != nil {
		h.logger.Error("Failed to search user", "error", err, "search", search)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Convert response
	response := h.convertGetUserResponse(resp)
	pkghttp.Success(c, http.StatusOK, resp.Message, response)
}

// Helper methods for response conversion
func (h *Handler) convertGetUsersResponse(resp *adminpb.GetUsersResponse) GetUsersResponse {
	users := make([]UserSummaryResponse, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = UserSummaryResponse{
			ID:           user.Id,
			Email:        user.Email,
			Phone:        user.Phone,
			Verified:     user.Verified,
			IsActive:     user.IsActive,
			IsPremium:    user.IsPremium,
			PremiumUntil: timeFromProto(user.PremiumUntil),
			LastLoginAt:  timeFromProto(user.LastLoginAt),
			CreatedAt:    user.CreatedAt.AsTime(),
			UpdatedAt:    user.UpdatedAt.AsTime(),
		}
	}

	return GetUsersResponse{
		Success: resp.Success,
		Message: resp.Message,
		Users:   users,
		Pagination: PaginationResponse{
			Total:       resp.Pagination.Total,
			Limit:       resp.Pagination.Limit,
			Offset:      resp.Pagination.Offset,
			HasMore:     resp.Pagination.HasMore,
			TotalPages:  resp.Pagination.TotalPages,
			CurrentPage: resp.Pagination.CurrentPage,
		},
	}
}

func (h *Handler) convertGetUserResponse(resp *adminpb.GetUserResponse) GetUserResponse {
	response := GetUserResponse{
		Success: resp.Success,
		Message: resp.Message,
		User: UserDetailsResponse{
			Auth: AuthDataResponse{
				ID:           resp.User.Auth.Id,
				Email:        resp.User.Auth.Email,
				Phone:        resp.User.Auth.Phone,
				Verified:     resp.User.Auth.Verified,
				IsActive:     resp.User.Auth.IsActive,
				IsPremium:    resp.User.Auth.IsPremium,
				PremiumUntil: timeFromProto(resp.User.Auth.PremiumUntil),
				LastLoginAt:  timeFromProto(resp.User.Auth.LastLoginAt),
				CreatedAt:    resp.User.Auth.CreatedAt.AsTime(),
				UpdatedAt:    resp.User.Auth.UpdatedAt.AsTime(),
			},
		},
	}

	// Convert profile if present
	if resp.User.Profile != nil {
		profile := &DetailedProfileDataResponse{
			ID:                    resp.User.Profile.Id,
			IsBride:               resp.User.Profile.IsBride,
			FullName:              resp.User.Profile.FullName,
			DateOfBirth:           resp.User.Profile.DateOfBirth,
			HeightCm:              resp.User.Profile.HeightCm,
			PhysicallyChallenged:  resp.User.Profile.PhysicallyChallenged,
			Community:             resp.User.Profile.Community,
			MaritalStatus:         resp.User.Profile.MaritalStatus,
			Profession:            resp.User.Profile.Profession,
			ProfessionType:        resp.User.Profile.ProfessionType,
			HighestEducationLevel: resp.User.Profile.HighestEducationLevel,
			HomeDistrict:          resp.User.Profile.HomeDistrict,
			ProfilePictureURL:     resp.User.Profile.ProfilePictureUrl,
			LastLogin:             timeFromProto(resp.User.Profile.LastLogin),
			Age:                   resp.User.Profile.Age,
		}

		// Convert partner preferences
		if resp.User.Profile.PartnerPreferences != nil {
			profile.PartnerPreferences = &PartnerPreferencesResponse{
				MinAgeYears:                resp.User.Profile.PartnerPreferences.MinAgeYears,
				MaxAgeYears:                resp.User.Profile.PartnerPreferences.MaxAgeYears,
				MinHeightCm:                resp.User.Profile.PartnerPreferences.MinHeightCm,
				MaxHeightCm:                resp.User.Profile.PartnerPreferences.MaxHeightCm,
				AcceptPhysicallyChallenged: resp.User.Profile.PartnerPreferences.AcceptPhysicallyChallenged,
				PreferredCommunities:       resp.User.Profile.PartnerPreferences.PreferredCommunities,
				PreferredMaritalStatus:     resp.User.Profile.PartnerPreferences.PreferredMaritalStatus,
				PreferredProfessions:       resp.User.Profile.PartnerPreferences.PreferredProfessions,
				PreferredProfessionTypes:   resp.User.Profile.PartnerPreferences.PreferredProfessionTypes,
				PreferredEducationLevels:   resp.User.Profile.PartnerPreferences.PreferredEducationLevels,
				PreferredHomeDistricts:     resp.User.Profile.PartnerPreferences.PreferredHomeDistricts,
			}
		}

		// Convert photos
		photos := make([]UserPhotoResponse, len(resp.User.Profile.AdditionalPhotos))
		for i, photo := range resp.User.Profile.AdditionalPhotos {
			photos[i] = UserPhotoResponse{
				PhotoURL:     photo.PhotoUrl,
				DisplayOrder: photo.DisplayOrder,
				CreatedAt:    photo.CreatedAt.AsTime(),
			}
		}
		profile.AdditionalPhotos = photos

		// Convert video
		if resp.User.Profile.IntroVideo != nil {
			profile.IntroVideo = &UserVideoResponse{
				VideoURL:        resp.User.Profile.IntroVideo.VideoUrl,
				FileName:        resp.User.Profile.IntroVideo.FileName,
				FileSize:        resp.User.Profile.IntroVideo.FileSize,
				DurationSeconds: resp.User.Profile.IntroVideo.DurationSeconds,
				CreatedAt:       resp.User.Profile.IntroVideo.CreatedAt.AsTime(),
			}
		}

		response.User.Profile = profile
	}

	return response
}

// Helper function to convert protobuf timestamp to Go time
func timeFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
