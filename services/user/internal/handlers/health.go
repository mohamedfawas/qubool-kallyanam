// services/user/internal/handlers/health.go
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// HealthHandler manages health check endpoints
type HealthHandler struct {
	reporter health.Reporter
	logger   logging.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	pgClient *postgres.Client,
	redisClient *redis.Client,
	mongoClient *mongodb.Client,
	logger logging.Logger,
) *HealthHandler {
	// Create health reporter
	reporter := health.NewReporter(
		"user-service",
		health.WithEnvironment("development"),
	)

	// Add PostgreSQL health check
	reporter.AddCheck(health.NewSimpleCheck(
		"postgres",
		health.TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			err := pgClient.Ping(ctx)
			details := map[string]interface{}{}
			if stats := pgClient.Stats(); stats != nil {
				details["stats"] = stats
			}
			return err == nil, details, err
		},
	))

	// Add Redis health check
	reporter.AddCheck(health.NewSimpleCheck(
		"redis",
		health.TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			err := redisClient.Ping(ctx)
			details := map[string]interface{}{}
			if stats := redisClient.Stats(); stats != nil {
				details["stats"] = stats
			}
			return err == nil, details, err
		},
	))

	// Add MongoDB health check
	reporter.AddCheck(health.NewSimpleCheck(
		"mongodb",
		health.TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			err := mongoClient.Ping(ctx)
			details := map[string]interface{}{}
			if stats := mongoClient.Stats(); stats != nil {
				details["stats"] = stats
			}
			return err == nil, details, err
		},
	))

	return &HealthHandler{
		reporter: reporter,
		logger:   logger,
	}
}

// Check handles the health check HTTP request
func (h *HealthHandler) Check(c *gin.Context) {
	// Set timeout for health checks
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Run all health checks
	status := h.reporter.RunChecks(ctx)

	// Set HTTP status code based on health status
	httpStatus := http.StatusOK
	if status.Status == health.StatusDown {
		httpStatus = http.StatusServiceUnavailable
	} else if status.Status == health.StatusDegraded {
		httpStatus = http.StatusTooManyRequests
	}

	c.JSON(httpStatus, status)
}

// LivenessCheck checks if the service is running
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// Run only liveness checks
	status := h.reporter.RunChecksFiltered(ctx, health.TypeLiveness)

	httpStatus := http.StatusOK
	if status.Status == health.StatusDown {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}

// ReadinessCheck checks if the service is ready to handle requests
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Run only readiness checks
	status := h.reporter.RunChecksFiltered(ctx, health.TypeReadiness)

	httpStatus := http.StatusOK
	if status.Status == health.StatusDown {
		httpStatus = http.StatusServiceUnavailable
	} else if status.Status == health.StatusDegraded {
		httpStatus = http.StatusTooManyRequests
	}

	c.JSON(httpStatus, status)
}
