package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
)

// AdminLogin handles admin authentication
func (h *Handler) AdminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid admin login request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate email
	if !validation.ValidateEmail(req.Email) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid email format", nil))
		return
	}

	// Call auth service with admin login
	success, accessToken, refreshToken, message, expiresIn, err := h.authClient.AdminLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("Admin login failed", "error", err, "email", req.Email)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	response := LoginResponse{
		Success:      success,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Message:      message,
	}

	// Return success response with 200 OK status
	pkghttp.Success(c, http.StatusOK, message, response)
}
