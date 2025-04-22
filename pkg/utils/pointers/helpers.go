package pointers

// String returns a pointer to the string value
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer or empty string if the pointer is nil
func StringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// StringValueOrDefault returns the value of the string pointer or the default value if the pointer is nil
func StringValueOrDefault(v *string, defaultValue string) string {
	if v == nil {
		return defaultValue
	}
	return *v
}

// Int returns a pointer to the int value
func Int(v int) *int {
	return &v
}

// IntValue returns the value of the int pointer or 0 if the pointer is nil
func IntValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// IntValueOrDefault returns the value of the int pointer or the default value if the pointer is nil
func IntValueOrDefault(v *int, defaultValue int) int {
	if v == nil {
		return defaultValue
	}
	return *v
}

// Bool returns a pointer to the bool value
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of the bool pointer or false if the pointer is nil
func BoolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// BoolValueOrDefault returns the value of the bool pointer or the default value if the pointer is nil
func BoolValueOrDefault(v *bool, defaultValue bool) bool {
	if v == nil {
		return defaultValue
	}
	return *v
}

// Float64 returns a pointer to the float64 value
func Float64(v float64) *float64 {
	return &v
}

// Float64Value returns the value of the float64 pointer or 0 if the pointer is nil
func Float64Value(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// Float64ValueOrDefault returns the value of the float64 pointer or the default value if the pointer is nil
func Float64ValueOrDefault(v *float64, defaultValue float64) float64 {
	if v == nil {
		return defaultValue
	}
	return *v
}

// Map creates a new map with the contents of the input map
// This is useful for creating a copy of a map that can be safely modified
func Map[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// Slice creates a new slice with the contents of the input slice
// This is useful for creating a copy of a slice that can be safely modified
func Slice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	result := make([]T, len(s))
	copy(result, s)
	return result
}
