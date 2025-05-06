// File: gateway/internal/handlers/v1/auth/login.go
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
)

// LoginRequest defines the request body for login
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse defines the response body for login
type LoginResponse struct {
	Success      bool   `json:"success"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	Message      string `json:"message"`
}

// Login handles user authentication
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid login request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate email
	if !validation.ValidateEmail(req.Email) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid email format", nil))
		return
	}

	// Call auth service
	success, accessToken, refreshToken, message, expiresIn, err := h.authClient.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("Login failed", "error", err, "email", req.Email)
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
