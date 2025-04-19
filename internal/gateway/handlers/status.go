// internal/gateway/handlers/status.go
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"go.uber.org/zap"
)

// StatusHandler handles gateway status endpoints
type StatusHandler struct {
	logger   *zap.Logger
	registry *discovery.ServiceRegistry
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(logger *zap.Logger, registry *discovery.ServiceRegistry) *StatusHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &StatusHandler{
		logger:   logger,
		registry: registry,
	}
}

// Register registers the status endpoints
func (h *StatusHandler) Register(router *gin.Engine) {
	router.GET("/status", h.GetStatus)
	router.GET("/status/services", h.GetServicesStatus)
}

// GetStatus returns the overall status of the gateway
func (h *StatusHandler) GetStatus(c *gin.Context) {
	services := h.registry.ListAll()

	// Check if all services are healthy
	allHealthy := true
	healthyCount := 0

	for _, service := range services {
		if service.Healthy {
			healthyCount++
		} else {
			allHealthy = false
		}
	}

	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}
	if healthyCount == 0 && len(services) > 0 {
		status = "critical"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    status,
		"total":     len(services),
		"healthy":   healthyCount,
		"unhealthy": len(services) - healthyCount,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetServicesStatus returns the status of all services
func (h *StatusHandler) GetServicesStatus(c *gin.Context) {
	services := h.registry.ListAll()

	// Simplify the response for clients
	response := make([]map[string]interface{}, 0, len(services))

	for _, service := range services {
		response = append(response, map[string]interface{}{
			"name":      service.Name,
			"host":      service.Host,
			"port":      service.Port,
			"type":      service.Type,
			"healthy":   service.Healthy,
			"last_seen": service.LastSeen.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}
