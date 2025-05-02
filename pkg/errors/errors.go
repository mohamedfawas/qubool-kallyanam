package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	BadRequest          ErrorType = "BAD_REQUEST"
	Unauthorized        ErrorType = "UNAUTHORIZED"
	Forbidden           ErrorType = "FORBIDDEN"
	NotFound            ErrorType = "NOT_FOUND"
	InternalServerError ErrorType = "INTERNAL_SERVER_ERROR"
)

// AppError represents application errors
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error returns the error message
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code for the error
func (e *AppError) StatusCode() int {
	switch e.Type {
	case BadRequest:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new AppError
func New(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// NewBadRequest creates a BadRequest error
func NewBadRequest(message string, err error) *AppError {
	return New(BadRequest, message, err)
}

// NewUnauthorized creates an Unauthorized error
func NewUnauthorized(message string, err error) *AppError {
	return New(Unauthorized, message, err)
}

// NewForbidden creates a Forbidden error
func NewForbidden(message string, err error) *AppError {
	return New(Forbidden, message, err)
}

// NewNotFound creates a NotFound error
func NewNotFound(message string, err error) *AppError {
	return New(NotFound, message, err)
}

// NewInternalServerError creates an InternalServerError
func NewInternalServerError(message string, err error) *AppError {
	return New(InternalServerError, message, err)
}
