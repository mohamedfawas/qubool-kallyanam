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
	StatusUnknown    Status = "UNKNOWN"
	StatusServing    Status = "SERVING"
	StatusNotServing Status = "NOT_SERVING"
)

// Checker is an interface for health checking a component
type Checker interface {
	Check(ctx context.Context) (Status, error)
	Name() string
}

// Service implements the gRPC health service
type Service struct {
	grpc_health_v1.UnimplementedHealthServer
	checkers   []Checker
	checkMutex sync.RWMutex
	timeout    time.Duration
}

// NewService creates a new health service
func NewService(timeout time.Duration, checkers ...Checker) *Service {
	return &Service{
		checkers: checkers,
		timeout:  timeout,
	}
}

// Check implements the gRPC health check protocol
func (s *Service) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	s.checkMutex.RLock()
	defer s.checkMutex.RUnlock()

	// If a specific service is requested
	if req.Service != "" {
		return s.checkService(ctx, req.Service)
	}

	// Check all services
	return s.checkAllServices(ctx)
}

// Watch implements the gRPC health check watch protocol (streaming)
func (s *Service) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// For MVP, just return a single status update
	resp, err := s.Check(stream.Context(), req)
	if err != nil {
		return err
	}
	return stream.Send(resp)
}

// AddChecker adds a health checker
func (s *Service) AddChecker(checker Checker) {
	s.checkMutex.Lock()
	defer s.checkMutex.Unlock()
	s.checkers = append(s.checkers, checker)
}

// checkService checks the health of a specific service
func (s *Service) checkService(ctx context.Context, serviceName string) (*grpc_health_v1.HealthCheckResponse, error) {
	for _, checker := range s.checkers {
		if checker.Name() == serviceName {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
			defer cancel()

			status, err := checker.Check(ctxWithTimeout)
			if err != nil {
				return &grpc_health_v1.HealthCheckResponse{
					Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
				}, nil
			}

			return &grpc_health_v1.HealthCheckResponse{
				Status: s.mapStatus(status),
			}, nil
		}
	}

	// Service not found
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN,
	}, nil
}

// checkAllServices checks the health of all registered services
func (s *Service) checkAllServices(ctx context.Context) (*grpc_health_v1.HealthCheckResponse, error) {
	// Consider the service healthy only if all checkers are healthy
	for _, checker := range s.checkers {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
		status, err := checker.Check(ctxWithTimeout)
		cancel()

		if err != nil || status != StatusServing {
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, nil
		}
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// mapStatus maps the internal status to the gRPC health check status
func (s *Service) mapStatus(status Status) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case StatusServing:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case StatusNotServing:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
}
