// services/gateway/internal/handlers/health.go
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients"
)

// HealthHandler manages health check endpoints
type HealthHandler struct {
	reporter     health.Reporter
	healthClient *clients.ServiceHealthClient
	logger       logging.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	healthClient *clients.ServiceHealthClient,
	logger logging.Logger,
) *HealthHandler {
	// Create health reporter
	reporter := health.NewReporter(
		"api-gateway",
		health.WithEnvironment("development"),
	)

	// Add downstream services health check
	reporter.AddCheck(health.NewSimpleCheck(
		"downstream-services",
		health.TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			serviceResults := healthClient.CheckAllServices(ctx)

			// Check if any service is down
			allHealthy := true
			for _, result := range serviceResults {
				if result.Status != health.StatusUp {
					allHealthy = false
					break
				}
			}

			return allHealthy, map[string]interface{}{"services": serviceResults}, nil
		},
	))

	return &HealthHandler{
		reporter:     reporter,
		healthClient: healthClient,
		logger:       logger,
	}
}

// Check handles the health check HTTP request
func (h *HealthHandler) Check(c *gin.Context) {
	// Set timeout for health checks
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
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

// LivenessCheck checks if the gateway is running
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Gateway liveness doesn't depend on downstream services
	c.JSON(http.StatusOK, health.ServiceStatus{
		Status:    health.StatusUp,
		Service:   "api-gateway",
		Timestamp: time.Now(),
		Components: map[string]health.Result{
			"gateway": {
				Name:      "gateway",
				Status:    health.StatusUp,
				Message:   "Gateway is running",
				Timestamp: time.Now(),
			},
		},
	})
}

// ReadinessCheck checks if the gateway and all downstream services are ready
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
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

// DetailedCheck provides detailed health information about all services
func (h *HealthHandler) DetailedCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	serviceResults := h.healthClient.CheckAllServices(ctx)

	// Determine overall status
	overallStatus := health.StatusUp
	for _, result := range serviceResults {
		if result.Status == health.StatusDown {
			overallStatus = health.StatusDown
			break
		} else if result.Status == health.StatusDegraded && overallStatus != health.StatusDown {
			overallStatus = health.StatusDegraded
		}
	}

	status := health.ServiceStatus{
		Status:     overallStatus,
		Service:    "api-gateway",
		Timestamp:  time.Now(),
		Components: serviceResults,
	}

	httpStatus := http.StatusOK
	if status.Status == health.StatusDown {
		httpStatus = http.StatusServiceUnavailable
	} else if status.Status == health.StatusDegraded {
		httpStatus = http.StatusTooManyRequests
	}

	c.JSON(httpStatus, status)
}
