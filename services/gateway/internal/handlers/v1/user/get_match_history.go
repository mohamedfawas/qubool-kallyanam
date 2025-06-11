package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// GetMatchHistory retrieves user's match history
func (h *Handler) GetMatchHistory(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse query parameters with defaults
	status := c.DefaultQuery("status", "")
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

	success, message, matches, pagination, err := h.userClient.GetMatchHistory(
		c.Request.Context(),
		userID.(string),
		status,
		limit,
		offset,
	)

	if err != nil {
		h.logger.Error("Failed to get match history", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	// Transform the matches for the response
	responseMatches := make([]gin.H, len(matches))
	for i, match := range matches {
		var actionDate time.Time
		if match.ActionDate != nil {
			actionDate = match.ActionDate.AsTime()
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
			"action":                  match.Action,
			"action_date":             actionDate,
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
