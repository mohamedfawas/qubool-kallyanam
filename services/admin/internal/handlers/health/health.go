package health

import (
	"context"
)

// HealthHandler provides health check functionality without database
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check returns service health status
func (h *HealthHandler) Check(ctx context.Context) error {
	// Simple health check - just return success since we have no database
	return nil
}

// Ready returns service readiness status
func (h *HealthHandler) Ready(ctx context.Context) error {
	// Could check if external services are reachable
	return nil
}
