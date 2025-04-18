package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests for the gateway service
type HealthHandler struct {
	logger *zap.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// Register registers the health check routes
func (h *HealthHandler) Register(router *gin.Engine) {
	router.GET("/health", h.Health)
}

// Health handles the health check request
func (h *HealthHandler) Health(c *gin.Context) {
	h.logger.Debug("Health check request received")

	// In a more complex implementation, check dependencies here

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "gateway",
	})
}
