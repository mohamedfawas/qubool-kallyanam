package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

func (h *Handler) GetMutualMatches(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse pagination parameters
	limit := 20
	offset := 0
	var err error

	limitParam := c.DefaultQuery("limit", "20")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit < 1 {
			limit = 20
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

	// Call user service
	success, message, matches, pagination, err := h.userClient.GetMutualMatches(
		c.Request.Context(),
		userID.(string),
		limit,
		offset,
	)

	if err != nil {
		h.logger.Error("Failed to get mutual matches", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	// Convert response
	responseMatches := make([]gin.H, len(matches))
	for i, match := range matches {
		var lastLogin time.Time
		if match.LastLogin != nil {
			lastLogin = match.LastLogin.AsTime()
		}

		var matchedAt time.Time
		if match.MatchedAt != nil {
			matchedAt = match.MatchedAt.AsTime()
		}

		responseMatches[i] = gin.H{
			"profile_id":              match.ProfileId,
			"full_name":               match.FullName,
			"age":                     match.Age,
			"height_cm":               match.HeightCm,
			"physically_challenged":   match.PhysicallyChallenged,
			"community":               match.Community,
			"marital_status":          match.MaritalStatus,
			"profession":              match.Profession,
			"profession_type":         match.ProfessionType,
			"highest_education_level": match.HighestEducationLevel,
			"home_district":           match.HomeDistrict,
			"profile_picture_url":     match.ProfilePictureUrl,
			"last_login":              lastLogin,
			"matched_at":              matchedAt,
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"matches": responseMatches,
		"pagination": gin.H{
			"total":    pagination.Total,
			"limit":    pagination.Limit,
			"offset":   pagination.Offset,
			"has_more": pagination.HasMore,
		},
	})
}
