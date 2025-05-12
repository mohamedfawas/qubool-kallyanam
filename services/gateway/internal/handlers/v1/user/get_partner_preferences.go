package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

func (h *Handler) GetPartnerPreferences(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	success, message, preferences, err := h.userClient.GetPartnerPreferences(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		h.logger.Error("Failed to get partner preferences", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":     success,
		"preferences": preferences,
	})
}
