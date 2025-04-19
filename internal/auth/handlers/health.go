package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests for the auth service
type HealthHandler struct {
	logger      *zap.Logger
	pgClient    *postgres.Client
	redisClient *redis.Client
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(logger *zap.Logger, pgClient *postgres.Client, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		logger:      logger,
		pgClient:    pgClient,
		redisClient: redisClient,
	}
}

// Register registers the health check routes
func (h *HealthHandler) Register(router *gin.Engine) {
	router.GET("/health", h.Health)
}

// Health handles the health check request
func (h *HealthHandler) Health(c *gin.Context) {
	h.logger.Debug("Health check request received")

	// Check database connections
	pgError := h.pgClient.Ping()
	redisError := h.redisClient.Ping(c.Request.Context())

	status := "ok"
	statusCode := http.StatusOK
	dbStatus := "ok"
	cacheStatus := "ok"

	if pgError != nil {
		h.logger.Error("Database health check failed", zap.Error(pgError))
		dbStatus = "error"
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	if redisError != nil {
		h.logger.Error("Redis health check failed", zap.Error(redisError))
		cacheStatus = "error"
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":  status,
		"service": "auth",
		"version": "v1.0.0",
		"components": gin.H{
			"database": dbStatus,
			"cache":    cacheStatus,
		},
	})
}
