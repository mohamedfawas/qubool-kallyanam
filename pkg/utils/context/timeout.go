package context

import (
	"context"
	"time"
)

// DefaultTimeout is the default timeout duration for operations
const DefaultTimeout = 30 * time.Second

// WithTimeout wraps context.WithTimeout but uses a default timeout if none specified
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// WithShortTimeout returns a context with a short timeout for quick operations
func WithShortTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}

// WithMediumTimeout returns a context with a medium timeout for standard operations
func WithMediumTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 15*time.Second)
}

// WithLongTimeout returns a context with a long timeout for operations that may take longer
func WithLongTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 60*time.Second)
}

// IsContextError returns true if the error is caused by context cancellation or deadline
func IsContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

// WithBackgroundContext creates a new background context with the same values as the original
// This is useful when you need to perform operations beyond the lifetime of the original context
func WithBackgroundContext(ctx context.Context, keys ...Key) context.Context {
	bgCtx := context.Background()

	// If no keys specified, copy all string values
	if len(keys) == 0 {
		// Copy common keys
		commonKeys := []Key{
			KeyUserID, KeyProfileID, KeyRequestID, KeyCorrelationID,
			KeyClientIP, KeyUserAgent, KeyUserRole, KeySessionID,
		}

		for _, k := range commonKeys {
			if v, ok := GetString(ctx, k); ok {
				bgCtx = WithValue(bgCtx, k, v)
			}
		}
		return bgCtx
	}

	// Copy only specified keys
	for _, k := range keys {
		if v := ctx.Value(k); v != nil {
			bgCtx = context.WithValue(bgCtx, k, v)
		}
	}

	return bgCtx
}

// WithValues creates a new context with multiple values added at once
func WithValues(ctx context.Context, values map[Key]interface{}) context.Context {
	for k, v := range values {
		ctx = WithValue(ctx, k, v)
	}
	return ctx
}
