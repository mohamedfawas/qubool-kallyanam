package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Response represents a standard API response
type Response struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Error      *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Success sends a successful response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	response := Response{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}

	c.JSON(statusCode, response)
}

// Error sends an error response
func Error(c *gin.Context, err error) {
	var statusCode int
	var errorType string
	var message string

	if appErr, ok := err.(*errors.AppError); ok {
		statusCode = appErr.StatusCode()
		errorType = string(appErr.Type)
		message = appErr.Message
	} else {
		statusCode = http.StatusInternalServerError
		errorType = string(errors.InternalServerError)
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

// StatusAccepted represents 202 Accepted status
const StatusAccepted = http.StatusAccepted

// NewBadRequest creates a bad request error
func NewBadRequest(message string, err error) *errors.AppError {
	return errors.NewBadRequest(message, err)
}

// FromGRPCError converts gRPC error to AppError
func FromGRPCError(err error) *errors.AppError {
	st, ok := status.FromError(err)
	if !ok {
		return errors.NewInternalServerError("Internal server error", err)
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return errors.NewBadRequest(st.Message(), err)
	case codes.AlreadyExists:
		return errors.NewBadRequest(st.Message(), err)
	case codes.NotFound:
		return errors.NewNotFound(st.Message(), err)
	case codes.Unauthenticated:
		return errors.NewUnauthorized(st.Message(), err)
	case codes.PermissionDenied:
		return errors.NewForbidden(st.Message(), err)
	default:
		return errors.NewInternalServerError("Internal server error", err)
	}
}
