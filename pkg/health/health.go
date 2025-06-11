package health

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"
)

// Status represents the health status of a component
type Status string

const (
	// StatusUnknown means we don't know the health state
	StatusUnknown Status = "UNKNOWN"
	// StatusServing means the component is running and healthy.
	StatusServing Status = "SERVING"
	// StatusNotServing means the component is running but not healthy.
	StatusNotServing Status = "NOT_SERVING"
)

// Checker is an interface for health checking a component.
// Any component that wants health checking should implement these two methods.
// For example, a database connection checker might implement:
//
//	func (db *DBChecker) Check(ctx context.Context) (Status, error) { ... }
//	func (db *DBChecker) Name() string { return "database" }
type Checker interface {
	Check(ctx context.Context) (Status, error)
	Name() string
}

// Service implements the gRPC health server by delegating to multiple Checkers.
// It holds a list of Checker instances and a timeout for each check.
type Service struct {
	grpc_health_v1.UnimplementedHealthServer // Embedding for default implementations

	checkers   []Checker     // List of components to check
	checkMutex sync.RWMutex  // Mutex to protect checkers slice for safe concurrent access
	timeout    time.Duration // Timeout for each individual health check
}

// NewService creates a new health Service.
// timeout: how long to wait for each check before considering it failed.
// checkers: one or more Checker implementations to register.
func NewService(timeout time.Duration, checkers ...Checker) *Service {
	return &Service{
		checkers: checkers,
		timeout:  timeout,
	}
}

// Check implements the unary gRPC health check endpoint.
// It checks either a specific service (if req.Service is non-empty) or all services.
func (s *Service) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Acquire read lock to allow multiple concurrent checks safely.
	s.checkMutex.RLock()
	defer s.checkMutex.RUnlock()

	// If the client asked for a specific service by name:
	if req.Service != "" {
		return s.checkService(ctx, req.Service)
	}

	// Otherwise, check all registered services
	return s.checkAllServices(ctx)
}

// Watch implements the streaming gRPC health check endpoint.
// Clients can subscribe to health updates. For now, we send only one update.
func (s *Service) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// For MVP, just return a single status update
	// Reuse our Check logic to get a single snapshot
	resp, err := s.Check(stream.Context(), req)
	if err != nil {
		return err // Return error if the check failed
	}
	// Send the response back over the stream.
	return stream.Send(resp)
}

// AddChecker allows adding a new Checker at runtime.
func (s *Service) AddChecker(checker Checker) {
	// Acquire write lock since we're modifying the slice
	s.checkMutex.Lock()
	defer s.checkMutex.Unlock()
	s.checkers = append(s.checkers, checker)
}

// checkService checks health for a single named service.
// If the service is not found, it returns SERVICE_UNKNOWN.
func (s *Service) checkService(ctx context.Context, serviceName string) (*grpc_health_v1.HealthCheckResponse, error) {
	// Loop through registered checkers to find the matching name
	for _, checker := range s.checkers {
		if checker.Name() == serviceName {
			// Create a child context with timeout for this check
			ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
			defer cancel()

			// Perform the actual health check
			status, err := checker.Check(ctxWithTimeout)
			if err != nil {
				// On error (e.g., timeout), report NOT_SERVING
				return &grpc_health_v1.HealthCheckResponse{
					Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
				}, nil
			}

			// Map our internal Status to gRPC enum and return
			return &grpc_health_v1.HealthCheckResponse{
				Status: s.mapStatus(status),
			}, nil
		}
	}

	// If no checker matches the given service name:
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN,
	}, nil
}

// checkAllServices checks all registered services in sequence.
// If any service is not serving, we return NOT_SERVING immediately
func (s *Service) checkAllServices(ctx context.Context) (*grpc_health_v1.HealthCheckResponse, error) {
	// Example: If we have a health checker for "db" and "cache",
	// we check database first, then cache.
	for _, checker := range s.checkers {
		// Create a timeout for each check individually.
		ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
		// Perform the check
		status, err := checker.Check(ctxWithTimeout)
		cancel() // Release resources immediately after check

		// If any checker returns an error or is not serving,
		// the overall service is considered NOT_SERVING.
		if err != nil || status != StatusServing {
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, nil
		}
	}

	// If all checks passed, the service is SERVING.
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// mapStatus converts our internal Status type to the gRPC ServingStatus enum
func (s *Service) mapStatus(status Status) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case StatusServing:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case StatusNotServing:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		// Covers StatusUnknown and any future statuses
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
}
