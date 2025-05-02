package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	authclient "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
)

// Handler handles auth-related HTTP requests
type Handler struct {
	authClient *authclient.Client
}

// NewHandler creates a new auth handler
func NewHandler(authClient *authclient.Client) *Handler {
	return &Handler{
		authClient: authClient,
	}
}

// RegisterRoutes registers the auth routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/health", h.HealthCheck)
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/verify", h.Verify)
	}
}

// HealthCheck handles the health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	status, err := h.authClient.HealthCheck(c.Request.Context())
	if err != nil {
		pkghttp.Error(c, errors.NewInternalServerError("Failed to check auth service health", err))
		return
	}

	pkghttp.Success(c, http.StatusOK, "Auth service health check", gin.H{
		"status": status,
	})
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkghttp.Error(c, errors.NewBadRequest("Invalid request", err))
		return
	}

	resp, err := h.authClient.RegisterUser(c.Request.Context(), req.Email, req.Phone, req.Password)
	if err != nil {
		pkghttp.Error(c, errors.NewInternalServerError("Failed to register user", err))
		return
	}

	if !resp.Success {
		pkghttp.Success(c, http.StatusBadRequest, resp.Message, gin.H{
			"error": resp.Error,
		})
		return
	}

	pkghttp.Success(c, http.StatusCreated, resp.Message, gin.H{
		"otp_expires_at":    resp.OtpExpiresAt,
		"resend_allowed_at": resp.ResendAllowedAt,
	})
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"` // Email or phone
		Password   string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkghttp.Error(c, errors.NewBadRequest("Invalid request", err))
		return
	}

	resp, err := h.authClient.Login(c.Request.Context(), req.Identifier, req.Password)
	if err != nil {
		pkghttp.Error(c, errors.NewInternalServerError("Failed to login", err))
		return
	}

	if !resp.Success {
		pkghttp.Success(c, http.StatusUnauthorized, resp.Message, gin.H{
			"error": resp.Error,
		})
		return
	}

	pkghttp.Success(c, http.StatusOK, resp.Message, gin.H{
		"access_token":     resp.AccessToken,
		"refresh_token":    resp.RefreshToken,
		"token_expires_at": resp.TokenExpiresAt,
	})
}

// Verify handles registration verification
func (h *Handler) Verify(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkghttp.Error(c, errors.NewBadRequest("Invalid request", err))
		return
	}

	resp, err := h.authClient.VerifyRegistration(c.Request.Context(), req.Email, req.OTP)
	if err != nil {
		pkghttp.Error(c, errors.NewInternalServerError("Failed to verify registration", err))
		return
	}

	if !resp.Success {
		pkghttp.Success(c, http.StatusBadRequest, resp.Message, gin.H{
			"error": resp.Error,
		})
		return
	}

	pkghttp.Success(c, http.StatusOK, resp.Message, gin.H{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
	})
}
