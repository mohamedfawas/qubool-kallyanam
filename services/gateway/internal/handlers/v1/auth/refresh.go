// File: gateway/internal/handlers/v1/auth/refresh.go
package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// RefreshTokenResponse defines the response body for token refresh
type RefreshTokenResponse struct {
	Success      bool   `json:"success"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	Message      string `json:"message"`
}

// RefreshToken handles token refresh requests
func (h *Handler) RefreshToken(c *gin.Context) {
	// Extract refresh token from Authorization header
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

	refreshToken := tokenParts[1]

	// Call auth service
	success, accessToken, newRefreshToken, expiresIn, message, err := h.authClient.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		h.logger.Error("Token refresh failed", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	response := RefreshTokenResponse{
		Success:      success,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    expiresIn,
		Message:      message,
	}

	// Return success response with 200 OK status
	pkghttp.Success(c, http.StatusOK, message, response)
}
