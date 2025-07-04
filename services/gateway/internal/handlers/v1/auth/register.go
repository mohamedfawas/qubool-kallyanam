package auth

import (
	"github.com/gin-gonic/gin"

	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/metrics"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
)

// RegisterRequest defines the request body for registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Handler handles HTTP requests for auth endpoints
type Handler struct {
	authClient *auth.Client
	logger     logging.Logger
	metrics    *metrics.Metrics
}

// NewHandler creates a new auth handler
func NewHandler(authClient *auth.Client, logger logging.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		authClient: authClient,
		logger:     logger,
		metrics:    metrics,
	}
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate email
	if !validation.ValidateEmail(req.Email) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid email format", nil))
		return
	}

	// Validate phone
	if !validation.ValidatePhone(req.Phone) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid phone format", nil))
		return
	}

	// Validate password
	if !validation.ValidatePassword(req.Password, validation.DefaultPasswordPolicy()) {
		pkghttp.Error(c, pkghttp.NewBadRequest("Password does not meet requirements", nil))
		return
	}

	// Call auth service
	success, message, err := h.authClient.Register(c.Request.Context(), req.Email, req.Phone, req.Password)
	if err != nil {
		h.logger.Error("Registration failed", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if success {
		h.metrics.IncrementUserRegistrations()
	}

	// Return success response with 202 Accepted status
	pkghttp.Success(c, pkghttp.StatusAccepted, message, gin.H{
		"success": success,
	})
}
