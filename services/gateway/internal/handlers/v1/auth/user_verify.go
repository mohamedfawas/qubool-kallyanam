// File: services/gateway/internal/handlers/v1/auth/verify.go

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
)

// VerifyRequest defines the request body for verification
type VerifyRequest struct {
	Email string `json:"email" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

// Verify handles OTP verification
func (h *Handler) Verify(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate email format
	if !validation.ValidateEmail(req.Email) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid email format", nil))
		return
	}

	// Call auth service to verify OTP
	success, message, err := h.authClient.Verify(c.Request.Context(), req.Email, req.OTP)
	if err != nil {
		h.logger.Error("Verification failed", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if success {
		h.metrics.IncrementUserVerifications()
	}

	// Return success response with 201 Created status
	pkghttp.Success(c, http.StatusCreated, message, gin.H{
		"success": success,
	})
}
