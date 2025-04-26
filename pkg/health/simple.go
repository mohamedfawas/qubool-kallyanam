// pkg/health/simple.go
package health

import (
	"context"
	"time"
)

// NewSimpleCheck creates a basic health checker with minimal configuration
func NewSimpleCheck(id string, checkType CheckType, fn CheckFn) Checker {
	return &SimpleCheck{
		id:        id,
		checkType: checkType,
		checkFn: func(ctx context.Context) (Status, string, map[string]interface{}) {
			healthy, details, err := fn(ctx)

			if details == nil {
				details = make(map[string]interface{})
			}

			if err != nil {
				return StatusDown, err.Error(), details
			}

			if healthy {
				return StatusUp, "Check passed", details
			}

			return StatusDown, "Check failed", details
		},
	}
}

// ID returns the checker identifier
func (c *SimpleCheck) ID() string {
	return c.id
}

// Type returns the check type
func (c *SimpleCheck) Type() CheckType {
	return c.checkType
}

// Check executes the health check
func (c *SimpleCheck) Check(ctx context.Context) Result {
	start := time.Now()

	// Execute the check function
	status, message, details := c.checkFn(ctx)

	// Record execution time
	duration := time.Since(start)

	if details == nil {
		details = make(map[string]interface{})
	}
	details["duration_ms"] = duration.Milliseconds()

	return Result{
		Name:      c.id,
		Status:    status,
		Message:   message,
		Details:   details,
		Timestamp: start,
		Duration:  duration,
	}
}

// NewDatabaseCheck creates a health checker for database connections
func NewDatabaseCheck(id string, timeout time.Duration, pingFn func(ctx context.Context) error) Checker {
	return NewSimpleCheck(
		id,
		TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			// Create context with timeout if needed
			execCtx := ctx
			if timeout > 0 {
				var cancel context.CancelFunc
				execCtx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			err := pingFn(execCtx)
			return err == nil, nil, err
		},
	)
}

// NewHTTPCheck creates a health checker for HTTP endpoints
func NewHTTPCheck(id string, url string, timeout time.Duration) Checker {
	// Implementation would make an HTTP request to the URL
	// For brevity, this is just a placeholder
	return NewSimpleCheck(
		id,
		TypeReadiness,
		func(ctx context.Context) (bool, map[string]interface{}, error) {
			// Create HTTP client with timeout and make request
			// This is simplified - actual implementation would use http.Client
			details := map[string]interface{}{
				"url": url,
			}
			return true, details, nil
		},
	)
}
