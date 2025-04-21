package logging

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// extractContextFields extracts known fields from context
func extractContextFields(ctx context.Context) []zapcore.Field {
	var fields []zapcore.Field

	// Extract request ID if present
	if v := ctx.Value(requestIDKey); v != nil {
		if id, ok := v.(string); ok && id != "" {
			fields = append(fields, zap.String("request_id", id))
		}
	}

	// Extract user ID if present
	if v := ctx.Value(userIDKey); v != nil {
		if id, ok := v.(string); ok && id != "" {
			fields = append(fields, zap.String("user_id", id))
		}
	}

	// Extract correlation ID if present
	if v := ctx.Value(correlationKey); v != nil {
		if id, ok := v.(string); ok && id != "" {
			fields = append(fields, zap.String("correlation_id", id))
		}
	}

	// Extract session ID if present
	if v := ctx.Value(sessionIDKey); v != nil {
		if id, ok := v.(string); ok && id != "" {
			fields = append(fields, zap.String("session_id", id))
		}
	}

	// Extract IP address if present
	if v := ctx.Value(ipAddressKey); v != nil {
		if ip, ok := v.(string); ok && ip != "" {
			fields = append(fields, zap.String("ip_address", ip))
		}
	}

	// Extract user agent if present
	if v := ctx.Value(userAgentKey); v != nil {
		if ua, ok := v.(string); ok && ua != "" {
			fields = append(fields, zap.String("user_agent", ua))
		}
	}

	return fields
}

// Context builder functions

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

// Common field constructors

// Error creates an error field
func Error(err error) zapcore.Field {
	return zap.Error(err)
}

// String creates a string field
func String(key, val string) zapcore.Field {
	return zap.String(key, val)
}

// Int creates an integer field
func Int(key string, val int) zapcore.Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) zapcore.Field {
	return zap.Int64(key, val)
}

// Float64 creates a float64 field
func Float64(key string, val float64) zapcore.Field {
	return zap.Float64(key, val)
}

// Bool creates a bool field
func Bool(key string, val bool) zapcore.Field {
	return zap.Bool(key, val)
}

// Duration creates a duration field
func Duration(key string, val time.Duration) zapcore.Field {
	return zap.Duration(key, val)
}

// Time creates a time field
func Time(key string, val time.Time) zapcore.Field {
	return zap.Time(key, val)
}

// Any creates a field with any value
func Any(key string, val interface{}) zapcore.Field {
	return zap.Any(key, val)
}

// Matrimony-specific field constructors

// ProfileID creates a profile ID field
func ProfileID(id string) zapcore.Field {
	return zap.String("profile_id", id)
}

// MatchID creates a match ID field
func MatchID(id string) zapcore.Field {
	return zap.String("match_id", id)
}

// ConversationID creates a conversation ID field
func ConversationID(id string) zapcore.Field {
	return zap.String("conversation_id", id)
}

// PaymentID creates a payment ID field
func PaymentID(id string) zapcore.Field {
	return zap.String("payment_id", id)
}

// EventName creates an event name field
func EventName(name string) zapcore.Field {
	return zap.String("event_name", name)
}

// Latency creates a latency field
func Latency(duration time.Duration) zapcore.Field {
	return zap.Duration("latency_ms", duration)
}
