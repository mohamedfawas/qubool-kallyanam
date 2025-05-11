package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorType is a custom string type to represent various categories of errors in the application.
// Example: Instead of returning plain strings like "400 Bad Request", we use more readable constants.
type ErrorType string

const (
	BadRequest          ErrorType = "BAD_REQUEST"           // For 400 errors - Client sent bad input
	Unauthorized        ErrorType = "UNAUTHORIZED"          // For 401 errors - User is not authenticated
	Forbidden           ErrorType = "FORBIDDEN"             // For 403 errors - Authenticated but access denied
	NotFound            ErrorType = "NOT_FOUND"             // For 404 errors - Resource not found
	InternalServerError ErrorType = "INTERNAL_SERVER_ERROR" // For 500 errors - Unexpected server error

	// Reusing HTTP constants for convenience
	StatusAccepted = http.StatusAccepted // Equivalent to 202 status code
)

// AppError is a custom structure that wraps error details in a structured way
type AppError struct {
	Type    ErrorType
	Message string // Human-readable message
	Err     error  // Actual underlying error for debugging
}

// Error method makes AppError compatible with Go's built-in error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// StatusCode returns the appropriate HTTP status code based on the error type
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

// API Response structures

// Response defines a standard format for API responses
type Response struct {
	StatusCode int         `json:"status_code"`     // HTTP status code (e.g., 200, 400, 500)
	Message    string      `json:"message"`         // Brief message to client
	Data       interface{} `json:"data,omitempty"`  // Actual response data, if any (optional)
	Error      *ErrorInfo  `json:"error,omitempty"` // Error details, if any (optional)
}

// ErrorInfo gives additional details about the error in API responses
type ErrorInfo struct {
	Type    string `json:"type"`    // Example: BAD_REQUEST
	Message string `json:"message"` // Example: "Email is required"
}

// Error Constructor Functions

// NewError creates a new AppError
func NewError(errType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// NewBadRequest creates a BadRequest error
func NewBadRequest(message string, err error) *AppError {
	return NewError(BadRequest, message, err)
}

// NewUnauthorized creates an Unauthorized error
func NewUnauthorized(message string, err error) *AppError {
	return NewError(Unauthorized, message, err)
}

// NewForbidden creates a Forbidden error
func NewForbidden(message string, err error) *AppError {
	return NewError(Forbidden, message, err)
}

// NewNotFound creates a NotFound error
func NewNotFound(message string, err error) *AppError {
	return NewError(NotFound, message, err)
}

// NewInternalServerError creates an InternalServerError
func NewInternalServerError(message string, err error) *AppError {
	return NewError(InternalServerError, message, err)
}

// Response handler functions

// Success is used to send a successful HTTP response
// Example: Success(c, 200, "User created", userData)
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	response := Response{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}

	c.JSON(statusCode, response)
}

// Error is used to send a structured error response
// Example: Error(c, NewBadRequest("Missing name", nil))
func Error(c *gin.Context, err error) {
	var statusCode int
	var errorType string
	var message string

	// Check if the error is of type AppError
	if appErr, ok := err.(*AppError); ok {
		statusCode = appErr.StatusCode() // Convert to proper HTTP code
		errorType = string(appErr.Type)  // BAD_REQUEST, FORBIDDEN etc
		message = appErr.Message         // Human-friendly message
	} else {
		// If it's not an AppError, treat it as unexpected
		statusCode = http.StatusInternalServerError
		errorType = string(InternalServerError)
		message = "An unexpected error occurred"
	}

	response := Response{
		StatusCode: statusCode,
		Message:    "Request failed",
		Error: &ErrorInfo{
			Type:    errorType,
			Message: message,
		},
	}

	c.JSON(statusCode, response)
}

// FromGRPCError converts a gRPC error into a structured AppError
// This is useful when your service communicates with another service via gRPC.
// Example: If another microservice sends gRPC error "NOT_FOUND", this will convert it to HTTP 404.
func FromGRPCError(err error) *AppError {
	// Try to get status from gRPC error
	st, ok := status.FromError(err)
	if !ok {
		return NewInternalServerError("Internal server error", err)
	}

	// Map gRPC error codes to our custom AppErrors
	switch st.Code() {
	case codes.InvalidArgument:
		return NewBadRequest(st.Message(), err)
	case codes.AlreadyExists:
		return NewBadRequest(st.Message(), err)
	case codes.NotFound:
		return NewNotFound(st.Message(), err)
	case codes.Unauthenticated:
		return NewUnauthorized(st.Message(), err)
	case codes.PermissionDenied:
		return NewForbidden(st.Message(), err)
	default:
		return NewInternalServerError("Internal server error", err)
	}
}
