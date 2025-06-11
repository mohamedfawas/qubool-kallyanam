package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// UpdateMatchAction handles updating a user's previous action on a potential match
func (h *Handler) UpdateMatchAction(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req struct {
		ProfileID uint64 `json:"profile_id" binding:"required"`
		Action    string `json:"action" binding:"required,oneof=liked disliked passed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format. Action must be one of: liked, disliked, passed", err))
		return
	}

	success, message, isMutualMatch, wasMutualMatchBroken, err := h.userClient.UpdateMatchAction(
		c.Request.Context(),
		userID.(string),
		req.ProfileID,
		req.Action,
	)

	if err != nil {
		h.logger.Error("Failed to update match action", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":                 success,
		"is_mutual_match":         isMutualMatch,
		"was_mutual_match_broken": wasMutualMatchBroken,
	})
}
