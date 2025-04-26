// structured.go
package logging

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Common context keys for logging
type contextKey string

const (
	requestIDKey   contextKey = "request_id"
	userIDKey      contextKey = "user_id"
	correlationKey contextKey = "correlation_id"
	sessionIDKey   contextKey = "session_id"
	ipAddressKey   contextKey = "ip_address"
	userAgentKey   contextKey = "user_agent"
)

// Context helper functions

// WithRequestID adds a request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithUserID adds a user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithCorrelationID adds a correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationKey, correlationID)
}

// WithSessionID adds a session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// WithIPAddress adds an IP address to context
func WithIPAddress(ctx context.Context, ipAddress string) context.Context {
	return context.WithValue(ctx, ipAddressKey, ipAddress)
}

// WithUserAgent adds a user agent to context
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, userAgentKey, userAgent)
}

// Extract fields from context
func extractContextFields(ctx context.Context) []Field {
	var fields []Field

	addStringField := func(key contextKey, fieldName string) {
		if v, ok := ctx.Value(key).(string); ok && v != "" {
			fields = append(fields, String(fieldName, v))
		}
	}

	addStringField(requestIDKey, "request_id")
	addStringField(userIDKey, "user_id")
	addStringField(correlationKey, "correlation_id")
	addStringField(sessionIDKey, "session_id")
	addStringField(ipAddressKey, "ip_address")
	addStringField(userAgentKey, "user_agent")

	return fields
}

// Field constructors

// Error creates an error field
func Error(err error) Field {
	return zap.Error(err)
}

// String creates a string field
func String(key, val string) Field {
	return zap.String(key, val)
}

// Int creates an integer field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Float64 creates a float64 field
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Duration creates a duration field
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Time creates a time field
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// Any creates a field with any value
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Application-specific field constructors

// ProfileID creates a profile ID field
func ProfileID(id string) Field {
	return String("profile_id", id)
}

// MatchID creates a match ID field
func MatchID(id string) Field {
	return String("match_id", id)
}

// ConversationID creates a conversation ID field
func ConversationID(id string) Field {
	return String("conversation_id", id)
}

// PaymentID creates a payment ID field
func PaymentID(id string) Field {
	return String("payment_id", id)
}

// EventName creates an event name field
func EventName(name string) Field {
	return String("event_name", name)
}

// Latency creates a latency field
func Latency(duration time.Duration) Field {
	return Duration("latency_ms", duration)
}
