package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// RecordMatchAction handles the user's action on a potential match
func (h *Handler) RecordMatchAction(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req struct {
		ProfileID uint64 `json:"profile_id" binding:"required,min=1"`
		Action    string `json:"action" binding:"required,oneof=liked passed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format. Action must be one of: liked or passed", err))
		return
	}

	success, message, isMutualMatch, err := h.userClient.RecordMatchAction(
		c.Request.Context(),
		userID.(string),
		req.ProfileID,
		req.Action,
	)

	if err != nil {
		h.logger.Error("Failed to record match action", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if success {
		if req.Action == "liked" {
			h.metrics.IncrementMatchesLiked()

			if isMutualMatch {
				h.metrics.IncrementMutualMatches()
			}
		} else if req.Action == "passed" {
			h.metrics.IncrementMatchesPassed()
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":         success,
		"is_mutual_match": isMutualMatch,
	})
}
