// pkg/discovery/registry.go
package discovery

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Update ServiceInfo to include dependencies
type ServiceInfo struct {
	Name             string
	Host             string
	Port             int
	Type             string // "grpc" or "http"
	Routes           []string
	HealthCheck      string
	LastSeen         time.Time
	Healthy          bool
	Dependencies     []string        // List of service names this service depends on
	DependencyStatus map[string]bool // Status of each dependency
}

// ServiceRegistry manages service discovery
type ServiceRegistry struct {
	services map[string]*ServiceInfo
	mutex    sync.RWMutex
	logger   *zap.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(logger *zap.Logger) *ServiceRegistry {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ServiceRegistry{
		services: make(map[string]*ServiceInfo),
		logger:   logger,
	}
}

// Register adds a service to the registry
func (sr *ServiceRegistry) Register(service *ServiceInfo) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	service.LastSeen = time.Now()
	service.Healthy = true
	sr.services[service.Name] = service

	sr.logger.Info("Service registered",
		zap.String("service", service.Name),
		zap.String("host", service.Host),
		zap.Int("port", service.Port))

	return nil
}

// Unregister removes a service from the registry
func (sr *ServiceRegistry) Unregister(serviceName string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.services[serviceName]; !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	delete(sr.services, serviceName)
	sr.logger.Info("Service unregistered", zap.String("service", serviceName))

	return nil
}

// Get returns information about a registered service
func (sr *ServiceRegistry) Get(serviceName string) (*ServiceInfo, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	return service, nil
}

// ListAll returns all registered services
func (sr *ServiceRegistry) ListAll() []*ServiceInfo {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	services := make([]*ServiceInfo, 0, len(sr.services))
	for _, service := range sr.services {
		services = append(services, service)
	}

	return services
}

// UpdateStatus updates the health status of a service
func (sr *ServiceRegistry) UpdateStatus(serviceName string, healthy bool) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	service.Healthy = healthy
	service.LastSeen = time.Now()

	return nil
}

// StartHealthCheck starts periodic health checks for all services
func (sr *ServiceRegistry) StartHealthCheck(client *http.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			sr.checkServicesHealth(client)
		}
	}()
}

// checkServicesHealth performs health checks on all registered services
func (sr *ServiceRegistry) checkServicesHealth(client *http.Client) {
	sr.mutex.RLock()
	services := make([]*ServiceInfo, 0, len(sr.services))
	for _, service := range sr.services {
		services = append(services, service)
	}
	sr.mutex.RUnlock()

	for _, service := range services {
		if service.HealthCheck == "" {
			continue
		}

		go func(svc *ServiceInfo) {
			healthURL := fmt.Sprintf("http://%s:%d%s", svc.Host, svc.Port, svc.HealthCheck)

			req, err := http.NewRequest("GET", healthURL, nil)
			if err != nil {
				sr.logger.Error("Failed to create health check request",
					zap.String("service", svc.Name),
					zap.Error(err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			resp, err := client.Do(req)
			if err != nil {
				sr.logger.Warn("Health check failed",
					zap.String("service", svc.Name),
					zap.Error(err))
				sr.UpdateStatus(svc.Name, false)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				sr.logger.Warn("Service not healthy",
					zap.String("service", svc.Name),
					zap.Int("status", resp.StatusCode))
				sr.UpdateStatus(svc.Name, false)
				return
			}

			sr.UpdateStatus(svc.Name, true)
		}(service)
	}
}

// AddDependency adds a dependency to a service
func (sr *ServiceRegistry) AddDependency(serviceName, dependencyName string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Check if dependency already exists
	for _, dep := range service.Dependencies {
		if dep == dependencyName {
			return nil // Already added
		}
	}

	// Add dependency
	service.Dependencies = append(service.Dependencies, dependencyName)

	// Initialize status if it doesn't exist
	if service.DependencyStatus == nil {
		service.DependencyStatus = make(map[string]bool)
	}

	// Set initial status based on whether the dependency exists and is healthy
	depService, depExists := sr.services[dependencyName]
	if depExists {
		service.DependencyStatus[dependencyName] = depService.Healthy
	} else {
		service.DependencyStatus[dependencyName] = false
	}

	sr.logger.Info("Dependency added",
		zap.String("service", serviceName),
		zap.String("dependency", dependencyName))

	return nil
}

// RemoveDependency removes a dependency from a service
func (sr *ServiceRegistry) RemoveDependency(serviceName, dependencyName string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Find dependency
	var newDeps []string
	found := false
	for _, dep := range service.Dependencies {
		if dep != dependencyName {
			newDeps = append(newDeps, dep)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("dependency %s not found for service %s", dependencyName, serviceName)
	}

	// Update dependencies
	service.Dependencies = newDeps

	// Remove from status
	if service.DependencyStatus != nil {
		delete(service.DependencyStatus, dependencyName)
	}

	sr.logger.Info("Dependency removed",
		zap.String("service", serviceName),
		zap.String("dependency", dependencyName))

	return nil
}

// UpdateDependencyStatus updates the status of a dependency
func (sr *ServiceRegistry) UpdateDependencyStatus(serviceName, dependencyName string, healthy bool) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Check if this is a valid dependency
	found := false
	for _, dep := range service.Dependencies {
		if dep == dependencyName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("dependency %s not found for service %s", dependencyName, serviceName)
	}

	// Initialize map if needed
	if service.DependencyStatus == nil {
		service.DependencyStatus = make(map[string]bool)
	}

	// Update status
	service.DependencyStatus[dependencyName] = healthy

	return nil
}

// GetDependencyStatus returns the status of dependencies for a service
func (sr *ServiceRegistry) GetDependencyStatus(serviceName string) (map[string]bool, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Create a copy to avoid race conditions
	result := make(map[string]bool)
	for dep, status := range service.DependencyStatus {
		result[dep] = status
	}

	return result, nil
}
