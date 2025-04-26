// services/gateway/internal/clients/health.go
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// ServiceHealthClient checks health of downstream services
type ServiceHealthClient struct {
	httpClient *http.Client
	services   map[string]string
	logger     logging.Logger
}

// NewServiceHealthClient creates a client for checking service health
func NewServiceHealthClient(services map[string]string, logger logging.Logger) *ServiceHealthClient {
	return &ServiceHealthClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		services: services,
		logger:   logger,
	}
}

// CheckService checks the health of a single service
func (c *ServiceHealthClient) CheckService(ctx context.Context, serviceName, serviceURL string) health.Result {
	start := time.Now()
	healthURL := fmt.Sprintf("http://%s/health", serviceURL)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return health.Result{
			Name:      serviceName,
			Status:    health.StatusDown,
			Message:   fmt.Sprintf("Failed to create request: %v", err),
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return health.Result{
			Name:      serviceName,
			Status:    health.StatusDown,
			Message:   fmt.Sprintf("Failed to connect: %v", err),
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}
	defer resp.Body.Close()

	// Parse the service health status
	var serviceStatus health.ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&serviceStatus); err != nil {
		return health.Result{
			Name:      serviceName,
			Status:    health.StatusUnknown,
			Message:   fmt.Sprintf("Failed to parse response: %v", err),
			Timestamp: start,
			Duration:  time.Since(start),
		}
	}

	return health.Result{
		Name:      serviceName,
		Status:    serviceStatus.Status,
		Message:   fmt.Sprintf("Service status: %s", serviceStatus.Status),
		Details:   map[string]interface{}{"service_status": serviceStatus},
		Timestamp: start,
		Duration:  time.Since(start),
	}
}

// CheckAllServices checks health for all configured services
func (c *ServiceHealthClient) CheckAllServices(ctx context.Context) map[string]health.Result {
	results := make(map[string]health.Result)

	for name, url := range c.services {
		results[name] = c.CheckService(ctx, name, url)
	}

	return results
}
