package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// AdminLogout handles admin logout
func (h *Handler) AdminLogout(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		h.logger.Debug("Missing authorization header for admin logout")
		pkghttp.Error(c, pkghttp.NewBadRequest("Missing authorization header", nil))
		return
	}

	// Parse the token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		h.logger.Debug("Invalid authorization format for admin logout")
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid authorization format", nil))
		return
	}

	accessToken := tokenParts[1]

	// Call auth service to logout - we can reuse the existing logout method
	// as it handles token invalidation regardless of the token type/role
	_, _, err := h.authClient.Logout(c.Request.Context(), accessToken)
	if err != nil {
		h.logger.Error("Admin logout failed", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Return success response with 204 No Content status
	c.Status(http.StatusNoContent)
}
