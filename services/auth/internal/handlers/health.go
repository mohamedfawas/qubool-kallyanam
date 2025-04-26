// services/auth/internal/handlers/health.go
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/response"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/adapters"
)

// HealthHandler implements health check endpoints
type HealthHandler struct {
	reporter health.Reporter
	logger   logging.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	postgres *adapters.PostgresAdapter,
	redis *adapters.RedisAdapter,
	logger logging.Logger,
) *HealthHandler {
	// Create a new health reporter for auth service
	reporter := health.NewReporter("auth-service",
		health.WithVersion("1.0.0"),
		health.WithEnvironment("development"),
	)

	// Register PostgreSQL health check
	reporter.AddCheck(health.NewDatabaseCheck(
		"postgres",
		5*time.Second,
		postgres.Ping,
	))

	// Register Redis health check
	reporter.AddCheck(health.NewDatabaseCheck(
		"redis",
		3*time.Second,
		redis.Ping,
	))

	return &HealthHandler{
		reporter: reporter,
		logger:   logger.Named("health"),
	}
}

// Check handles the health check HTTP endpoint
func (h *HealthHandler) Check(c *gin.Context) {
	// Run all health checks
	serviceStatus := h.reporter.RunChecks(c.Request.Context())

	// Determine HTTP status code based on health status
	statusCode := http.StatusOK
	if serviceStatus.Status != health.StatusUp {
		statusCode = http.StatusServiceUnavailable
	}

	// Log health check results
	h.logger.Info("Health check executed",
		logging.String("status", string(serviceStatus.Status)),
		logging.Int("components", len(serviceStatus.Components)),
	)

	// Send response
	response.Send(c, statusCode, "Health check", serviceStatus)
}

// ReadinessCheck handles readiness probes (used in Kubernetes)
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// Run only readiness checks
	serviceStatus := h.reporter.RunChecksFiltered(c.Request.Context(), health.TypeReadiness)

	statusCode := http.StatusOK
	if serviceStatus.Status != health.StatusUp {
		statusCode = http.StatusServiceUnavailable
	}

	response.Send(c, statusCode, "Readiness check", serviceStatus)
}

// LivenessCheck handles liveness probes (used in Kubernetes)
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Run only liveness checks
	serviceStatus := h.reporter.RunChecksFiltered(c.Request.Context(), health.TypeLiveness)

	statusCode := http.StatusOK
	if serviceStatus.Status != health.StatusUp {
		statusCode = http.StatusServiceUnavailable
	}

	response.Send(c, statusCode, "Liveness check", serviceStatus)
}
