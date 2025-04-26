// pkg/health/health.go
package health

import (
	"context"
	"time"
)

// Status represents a component or service health state
type Status string

const (
	StatusUp       Status = "UP"       // Component is healthy and operational
	StatusDown     Status = "DOWN"     // Component is unhealthy or unavailable
	StatusDegraded Status = "DEGRADED" // Component is operational but with reduced functionality
	StatusUnknown  Status = "UNKNOWN"  // Component's health couldn't be determined
)

// CheckType indicates the check's purpose in a health system
type CheckType string

const (
	TypeLiveness  CheckType = "LIVENESS"  // Verifies the service is running
	TypeReadiness CheckType = "READINESS" // Verifies the service can handle requests
	TypeStartup   CheckType = "STARTUP"   // Verifies the service has completed initialization
)

// Result contains the outcome of a health check
type Result struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration_ms"` // In milliseconds for nicer JSON
}

// ServiceStatus represents the overall health of a service
type ServiceStatus struct {
	Status      Status            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Components  map[string]Result `json:"components,omitempty"`
}

// Checker defines a health check that can be executed
type Checker interface {
	// ID returns the unique identifier for this check
	ID() string

	// Type returns the check category
	Type() CheckType

	// Check performs the health verification
	Check(ctx context.Context) Result
}

// CheckerFunc is a function type that implements Checker
type CheckerFunc func(ctx context.Context) Result

// SimpleCheck provides an easy way to implement a health check
type SimpleCheck struct {
	id        string
	checkType CheckType
	checkFn   func(ctx context.Context) (Status, string, map[string]interface{})
}

// CheckFn is a simplified function for health checks
type CheckFn func(ctx context.Context) (healthy bool, details map[string]interface{}, err error)
