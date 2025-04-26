// pkg/health/reporter.go
package health

import (
	"context"
	"sync"
	"time"
)

// Reporter collects and executes health checks
type Reporter interface {
	// AddCheck registers a health check
	AddCheck(check Checker)

	// RemoveCheck unregisters a health check by ID
	RemoveCheck(id string)

	// RunChecks executes all registered health checks
	RunChecks(ctx context.Context) ServiceStatus

	// RunChecksFiltered runs only checks of the specified type
	RunChecksFiltered(ctx context.Context, checkType CheckType) ServiceStatus
}

// reporter implements the Reporter interface
type reporter struct {
	service     string
	version     string
	environment string
	checks      map[string]Checker
	mutex       sync.RWMutex
}

// NewReporter creates a new health check reporter
func NewReporter(service string, opts ...ReporterOption) Reporter {
	r := &reporter{
		service: service,
		checks:  make(map[string]Checker),
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ReporterOption configures a Reporter
type ReporterOption func(*reporter)

// WithVersion sets the service version
func WithVersion(version string) ReporterOption {
	return func(r *reporter) {
		r.version = version
	}
}

// WithEnvironment sets the deployment environment
func WithEnvironment(env string) ReporterOption {
	return func(r *reporter) {
		r.environment = env
	}
}

// AddCheck registers a health check
func (r *reporter) AddCheck(check Checker) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.checks[check.ID()] = check
}

// RemoveCheck unregisters a health check
func (r *reporter) RemoveCheck(id string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.checks, id)
}

// RunChecks performs all health checks
func (r *reporter) RunChecks(ctx context.Context) ServiceStatus {
	return r.runChecksInternal(ctx, nil)
}

// RunChecksFiltered runs only checks of the specified type
func (r *reporter) RunChecksFiltered(ctx context.Context, checkType CheckType) ServiceStatus {
	return r.runChecksInternal(ctx, &checkType)
}

// runChecksInternal executes health checks with optional filtering
func (r *reporter) runChecksInternal(ctx context.Context, filterType *CheckType) ServiceStatus {
	r.mutex.RLock()
	checksToRun := make([]Checker, 0, len(r.checks))
	for _, check := range r.checks {
		if filterType == nil || check.Type() == *filterType {
			checksToRun = append(checksToRun, check)
		}
	}
	r.mutex.RUnlock()

	// Fast path for no checks
	if len(checksToRun) == 0 {
		return ServiceStatus{
			Status:      StatusUnknown,
			Service:     r.service,
			Version:     r.version,
			Environment: r.environment,
			Timestamp:   time.Now(),
			Components:  make(map[string]Result),
		}
	}

	// Run checks concurrently
	results := make(map[string]Result)
	resultCh := make(chan Result, len(checksToRun))

	var wg sync.WaitGroup
	for _, check := range checksToRun {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			resultCh <- c.Check(ctx)
		}(check)
	}

	// Wait for completion and close channel
	wg.Wait()
	close(resultCh)

	// Process results
	overallStatus := StatusUp
	for result := range resultCh {
		results[result.Name] = result

		// Determine overall status (worst status wins)
		switch {
		case result.Status == StatusDown:
			overallStatus = StatusDown
		case result.Status == StatusDegraded && overallStatus != StatusDown:
			overallStatus = StatusDegraded
		case result.Status == StatusUnknown &&
			overallStatus != StatusDown &&
			overallStatus != StatusDegraded:
			overallStatus = StatusUnknown
		}
	}

	return ServiceStatus{
		Status:      overallStatus,
		Service:     r.service,
		Version:     r.version,
		Environment: r.environment,
		Timestamp:   time.Now(),
		Components:  results,
	}
}
