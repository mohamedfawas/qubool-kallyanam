// pkg/utils/context/request_id.go
package context

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type requestIDKey struct{}

// RequestIDKey is the context key for request ID
var RequestIDKey = requestIDKey{}

// GenerateRequestID generates a new random request ID
func GenerateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithRequestID adds a request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
