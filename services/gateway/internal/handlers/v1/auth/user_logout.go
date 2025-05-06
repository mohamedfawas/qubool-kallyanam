// File: services/gateway/internal/handlers/v1/auth/logout.go
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		h.logger.Debug("Missing authorization header")
		pkghttp.Error(c, pkghttp.NewBadRequest("Missing authorization header", nil))
		return
	}

	// Parse the token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		h.logger.Debug("Invalid authorization format")
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid authorization format", nil))
		return
	}

	accessToken := tokenParts[1]

	// Call auth service to logout
	_, _, err := h.authClient.Logout(c.Request.Context(), accessToken)
	if err != nil {
		h.logger.Error("Logout failed", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Return success response with 204 No Content status
	c.Status(http.StatusNoContent)
}
