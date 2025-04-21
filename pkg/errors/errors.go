package errors

import (
	"errors"
	"fmt"
)

// AppError represents a structured application error
type AppError struct {
	code    Code
	message string
	details map[string]interface{}
	cause   error
}

// New creates a new AppError
func New(code Code, message string) *AppError {
	return &AppError{
		code:    code,
		message: message,
		details: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with new context information
func Wrap(err error, code Code, message string) *AppError {
	if err == nil {
		return nil
	}

	// If the error is already an AppError, update its information
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			code:    code,
			message: message,
			details: appErr.details,
			cause:   appErr.cause,
		}
	}

	return &AppError{
		code:    code,
		message: message,
		details: make(map[string]interface{}),
		cause:   err,
	}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Code returns the error code
func (e *AppError) Code() Code {
	return e.code
}

// Cause returns the underlying cause of the error
func (e *AppError) Cause() error {
	return e.cause
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	e.details[key] = value
	return e
}

// WithDetails adds multiple details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	for k, v := range details {
		e.details[k] = v
	}
	return e
}

// Details returns all error details
func (e *AppError) Details() map[string]interface{} {
	return e.details
}

// GRPCCode returns the gRPC status code
func (e *AppError) GRPCCode() int {
	return e.code.ToGRPCCode()
}

// HTTPStatusCode returns the HTTP status code
func (e *AppError) HTTPStatusCode() int {
	return e.code.HTTPStatusCode()
}

// IsCode checks if error is of a specific code
func IsCode(err error, code Code) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.code == code
	}
	return false
}

// FromError creates an AppError from a standard error
// Returns nil if err is nil
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		code:    CodeUnknown,
		message: err.Error(),
		details: make(map[string]interface{}),
		cause:   err,
	}
}
