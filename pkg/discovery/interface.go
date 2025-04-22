package discovery

import (
	"context"
	"time"
)

// ServiceInstance represents a registered service instance
type ServiceInstance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	Secure    bool              `json:"secure"`
	Metadata  map[string]string `json:"metadata"`
	Healthy   bool              `json:"healthy"`
	CreatedAt time.Time         `json:"created_at"`
}

// ServiceQuery represents parameters for service discovery query
type ServiceQuery struct {
	// Service name to query
	Name string
	// Only include healthy instances
	OnlyHealthy bool
	// Tags to filter by (instances must have all specified tags)
	Tags []string
	// Timeout for the query
	Timeout time.Duration
}

// RegistrationOptions contains parameters for service registration
type RegistrationOptions struct {
	// How often the service should check in
	TTL time.Duration
	// Tags to associate with the service
	Tags []string
	// Additional metadata
	Metadata map[string]string
	// Whether this service supports TLS
	Secure bool
}

// Registry defines service discovery operations
type Registry interface {
	// Register registers a service instance
	Register(ctx context.Context, name, address string, port int, opts RegistrationOptions) (string, error)

	// Deregister removes a service instance from the registry
	Deregister(ctx context.Context, serviceID string) error

	// GetService finds all instances of a service
	GetService(ctx context.Context, query ServiceQuery) ([]ServiceInstance, error)

	// ReportHealthy reports a service instance as healthy
	ReportHealthy(ctx context.Context, serviceID string) error

	// Shutdown gracefully shuts down the registry client
	Shutdown(ctx context.Context) error
}
