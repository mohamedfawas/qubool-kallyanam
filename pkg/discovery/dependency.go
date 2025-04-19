// pkg/discovery/dependency.go

package discovery

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// DependencyChecker checks and updates dependency status
type DependencyChecker struct {
	registry *ServiceRegistry
	client   *http.Client
	logger   *zap.Logger
}

// NewDependencyChecker creates a new dependency checker
func NewDependencyChecker(registry *ServiceRegistry, logger *zap.Logger) *DependencyChecker {
	if logger == nil {
		logger = zap.NewNop()
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &DependencyChecker{
		registry: registry,
		client:   client,
		logger:   logger,
	}
}

// Start begins dependency checks
func (dc *DependencyChecker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			dc.checkAllDependencies()
		}
	}()
}

// checkAllDependencies checks all dependencies for all services
func (dc *DependencyChecker) checkAllDependencies() {
	services := dc.registry.ListAll()

	for _, service := range services {
		for _, depName := range service.Dependencies {
			dependency, err := dc.registry.Get(depName)
			if err != nil {
				dc.logger.Warn("Dependency not found in registry",
					zap.String("service", service.Name),
					zap.String("dependency", depName))

				// Update dependency status
				dc.registry.UpdateDependencyStatus(service.Name, depName, false)
				continue
			}

			// Update with the current health status of the dependency
			dc.registry.UpdateDependencyStatus(service.Name, depName, dependency.Healthy)
		}
	}
}

// RegisterDependency adds a dependency relationship
func (dc *DependencyChecker) RegisterDependency(serviceName, dependencyName string) error {
	service, err := dc.registry.Get(serviceName)
	if err != nil {
		return err
	}

	// Check if dependency already exists
	for _, dep := range service.Dependencies {
		if dep == dependencyName {
			return nil // Already registered
		}
	}

	// Add dependency
	return dc.registry.AddDependency(serviceName, dependencyName)
}
