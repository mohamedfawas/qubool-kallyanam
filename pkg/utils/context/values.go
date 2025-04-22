package context

import (
	"context"
)

// Key is a type for context keys
type Key string

// Common context keys
const (
	KeyUserID        Key = "user_id"
	KeyProfileID     Key = "profile_id"
	KeyRequestID     Key = "request_id"
	KeyCorrelationID Key = "correlation_id"
	KeyClientIP      Key = "client_ip"
	KeyUserAgent     Key = "user_agent"
	KeyUserRole      Key = "user_role"
	KeySessionID     Key = "session_id"
)

// WithValue adds a value to the context using a string key
func WithValue(ctx context.Context, key Key, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// GetString retrieves a string value from the context
func GetString(ctx context.Context, key Key) (string, bool) {
	value, ok := ctx.Value(key).(string)
	return value, ok
}

// MustGetString retrieves a string value from the context or returns default if not found
func MustGetString(ctx context.Context, key Key, defaultValue string) string {
	if value, ok := GetString(ctx, key); ok {
		return value
	}
	return defaultValue
}

// GetInt retrieves an int value from the context
func GetInt(ctx context.Context, key Key) (int, bool) {
	value, ok := ctx.Value(key).(int)
	return value, ok
}

// MustGetInt retrieves an int value from the context or returns default if not found
func MustGetInt(ctx context.Context, key Key, defaultValue int) int {
	if value, ok := GetInt(ctx, key); ok {
		return value
	}
	return defaultValue
}

// GetBool retrieves a bool value from the context
func GetBool(ctx context.Context, key Key) (bool, bool) {
	value, ok := ctx.Value(key).(bool)
	return value, ok
}

// MustGetBool retrieves a bool value from the context or returns default if not found
func MustGetBool(ctx context.Context, key Key, defaultValue bool) bool {
	if value, ok := GetBool(ctx, key); ok {
		return value
	}
	return defaultValue
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) (string, bool) {
	return GetString(ctx, KeyUserID)
}

// GetProfileID retrieves the profile ID from the context
func GetProfileID(ctx context.Context) (string, bool) {
	return GetString(ctx, KeyProfileID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) (string, bool) {
	return GetString(ctx, KeyRequestID)
}

// GetUserRole retrieves the user role from the context
func GetUserRole(ctx context.Context) (string, bool) {
	return GetString(ctx, KeyUserRole)
}
