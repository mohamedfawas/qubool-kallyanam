// internal/gateway/handlers/health.go

package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	logger      *zap.Logger
	redisClient *redis.Client
	registry    *discovery.ServiceRegistry
	// Add mutex for status history
	statusHistory     []SystemStatus
	statusHistoryMux  sync.RWMutex
	maxHistoryEntries int
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	Status       string                   `json:"status"`
	Timestamp    time.Time                `json:"timestamp"`
	Services     map[string]ServiceStatus `json:"services"`
	Dependencies map[string][]string      `json:"dependencies"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Healthy      bool            `json:"healthy"`
	LastSeen     time.Time       `json:"last_seen"`
	Dependencies map[string]bool `json:"dependencies"`
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(logger *zap.Logger, redisClient *redis.Client, registry *discovery.ServiceRegistry) *HealthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HealthHandler{
		logger:            logger,
		redisClient:       redisClient,
		registry:          registry,
		statusHistory:     make([]SystemStatus, 0, 100),
		maxHistoryEntries: 100, // Keep last 100 status entries
	}
}

// Register registers health check endpoints
func (h *HealthHandler) Register(router *gin.Engine) {
	// API endpoints
	router.GET("/health", h.GetHealth)
	router.GET("/health/details", h.GetHealthDetails)
	router.GET("/health/history", h.GetHealthHistory)

	// Dashboard UI
	router.GET("/dashboard", h.ServeDashboard)
	router.GET("/dashboard/data", h.GetDashboardData)

	// Start background task to update status history
	go h.recordStatusHistory()
}

// GetHealth handles basic health check
func (h *HealthHandler) GetHealth(c *gin.Context) {
	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisErr := h.redisClient.Ping(ctx)

	// Get service statuses
	services := h.registry.ListAll()
	allServicesHealthy := true

	for _, service := range services {
		if !service.Healthy {
			allServicesHealthy = false
			break
		}
	}

	status := "healthy"
	statusCode := http.StatusOK

	// Determine overall status
	if redisErr != nil && !allServicesHealthy {
		status = "critical"
		statusCode = http.StatusServiceUnavailable
	} else if redisErr != nil || !allServicesHealthy {
		status = "degraded"
		statusCode = http.StatusOK // Still return 200 for degraded to avoid cascading failures
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetHealthDetails returns detailed health information
func (h *HealthHandler) GetHealthDetails(c *gin.Context) {
	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisErr := h.redisClient.Ping(ctx)
	redisStatus := "healthy"
	if redisErr != nil {
		redisStatus = "unhealthy"
	}

	// Get service statuses
	services := h.registry.ListAll()
	serviceStatuses := make(map[string]ServiceStatus)
	dependencies := make(map[string][]string)

	for _, service := range services {
		// Create a copy of the dependency status
		dependencyStatus := make(map[string]bool)
		for dep, status := range service.DependencyStatus {
			dependencyStatus[dep] = status
		}

		serviceStatuses[service.Name] = ServiceStatus{
			Healthy:      service.Healthy,
			LastSeen:     service.LastSeen,
			Dependencies: dependencyStatus,
		}

		dependencies[service.Name] = service.Dependencies
	}

	// Calculate overall status
	overallStatus := "healthy"
	if redisErr != nil {
		overallStatus = "degraded"
	}

	criticalCount := 0
	for _, status := range serviceStatuses {
		if !status.Healthy {
			overallStatus = "degraded"
			criticalCount++
		}
	}

	if criticalCount >= len(serviceStatuses)/2 || (redisErr != nil && criticalCount > 0) {
		overallStatus = "critical"
	}

	response := SystemStatus{
		Status:       overallStatus,
		Timestamp:    time.Now(),
		Services:     serviceStatuses,
		Dependencies: dependencies,
	}

	// Include redisStatus in the response
	c.JSON(http.StatusOK, gin.H{
		"status":       response.Status,
		"timestamp":    response.Timestamp,
		"services":     response.Services,
		"dependencies": response.Dependencies,
		"redis_status": redisStatus,
	})
}

// GetHealthHistory returns status history
func (h *HealthHandler) GetHealthHistory(c *gin.Context) {
	h.statusHistoryMux.RLock()
	defer h.statusHistoryMux.RUnlock()

	// Return last 20 entries or all if fewer
	limit := 20
	if len(h.statusHistory) < limit {
		limit = len(h.statusHistory)
	}

	c.JSON(http.StatusOK, h.statusHistory[len(h.statusHistory)-limit:])
}

// recordStatusHistory periodically records system status
func (h *HealthHandler) recordStatusHistory() {
	ticker := time.NewTicker(60 * time.Second) // Record every minute
	defer ticker.Stop()

	for range ticker.C {
		// Get current status
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		redisErr := h.redisClient.Ping(ctx)
		cancel()

		services := h.registry.ListAll()
		serviceStatuses := make(map[string]ServiceStatus)
		dependencies := make(map[string][]string)

		allServicesHealthy := true
		for _, service := range services {
			if !service.Healthy {
				allServicesHealthy = false
			}

			// Create a copy of dependency status
			dependencyStatus := make(map[string]bool)
			for dep, status := range service.DependencyStatus {
				dependencyStatus[dep] = status
			}

			serviceStatuses[service.Name] = ServiceStatus{
				Healthy:      service.Healthy,
				LastSeen:     service.LastSeen,
				Dependencies: dependencyStatus,
			}

			dependencies[service.Name] = service.Dependencies
		}

		// Determine overall status
		status := "healthy"
		if redisErr != nil && !allServicesHealthy {
			status = "critical"
		} else if redisErr != nil || !allServicesHealthy {
			status = "degraded"
		}

		// Create system status record
		systemStatus := SystemStatus{
			Status:       status,
			Timestamp:    time.Now(),
			Services:     serviceStatuses,
			Dependencies: dependencies,
		}

		// Add to history
		h.statusHistoryMux.Lock()
		h.statusHistory = append(h.statusHistory, systemStatus)
		// Trim history if needed
		if len(h.statusHistory) > h.maxHistoryEntries {
			h.statusHistory = h.statusHistory[len(h.statusHistory)-h.maxHistoryEntries:]
		}
		h.statusHistoryMux.Unlock()
	}
}

// ServeDashboard serves the health dashboard UI
func (h *HealthHandler) ServeDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Service Health Dashboard",
	})
}

// GetDashboardData provides data for the dashboard
func (h *HealthHandler) GetDashboardData(c *gin.Context) {
	// Reuse GetHealthDetails to get the data
	h.GetHealthDetails(c)
}
