package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// GetRecommendedMatches retrieves potential matches for the user
func (h *Handler) GetRecommendedMatches(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse query parameters with defaults
	limit := 10
	offset := 0
	var err error

	limitParam := c.DefaultQuery("limit", "10")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit < 1 {
			limit = 10
		}
		// Cap max limit
		if limit > 50 {
			limit = 50
		}
	}

	offsetParam := c.DefaultQuery("offset", "0")
	if offsetParam != "" {
		offset, err = strconv.Atoi(offsetParam)
		if err != nil || offset < 0 {
			offset = 0
		}
	}

	success, message, profiles, pagination, err := h.userClient.GetRecommendedMatches(
		c.Request.Context(),
		userID.(string),
		limit,
		offset,
	)

	if err != nil {
		h.logger.Error("Failed to get recommended matches", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	// Transform the profiles for the response
	responseProfiles := make([]gin.H, len(profiles))
	for i, profile := range profiles {
		var lastLogin time.Time
		if profile.LastLogin != nil {
			lastLogin = profile.LastLogin.AsTime()
		}

		responseProfiles[i] = gin.H{
			"profile_id":              profile.ProfileId,
			"full_name":               profile.FullName,
			"age":                     profile.Age,
			"height_cm":               profile.HeightCm,
			"physically_challenged":   profile.PhysicallyChallenged,
			"community":               profile.Community,
			"marital_status":          profile.MaritalStatus,
			"profession":              profile.Profession,
			"profession_type":         profile.ProfessionType,
			"highest_education_level": profile.HighestEducationLevel,
			"home_district":           profile.HomeDistrict,
			"profile_picture_url":     profile.ProfilePictureUrl,
			"last_login":              lastLogin,
			"match_reasons":           profile.MatchReasons,
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"profiles": responseProfiles,
		"pagination": gin.H{
			"total":    pagination.Total,
			"limit":    pagination.Limit,
			"offset":   pagination.Offset,
			"has_more": pagination.HasMore,
		},
	})
}
