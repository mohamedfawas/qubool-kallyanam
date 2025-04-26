package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
)

// Response represents a standardized API response
type Response struct {
	StatusCode int         `json:"statuscode"`      // HTTP status code
	Message    string      `json:"message"`         // Human-readable message
	Data       interface{} `json:"data,omitempty"`  // Response data
	Error      *ErrorInfo  `json:"error,omitempty"` // Error details
}

// ErrorInfo contains error information
type ErrorInfo struct {
	Code    string                 `json:"code"`              // Error code
	Message string                 `json:"message"`           // Error message
	Details map[string]interface{} `json:"details,omitempty"` // Additional details
}

// Send sends a custom response
func Send(c *gin.Context, statusCode int, message string, data interface{}) {
	response := Response{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
	c.JSON(statusCode, response)
}

// SendError sends an error response
func SendError(c *gin.Context, err error, message string) {
	appErr := errors.FromError(err)
	statusCode := appErr.Code().HTTPStatusCode()

	errInfo := &ErrorInfo{
		Code:    string(appErr.Code()),
		Message: appErr.Error(),
		Details: appErr.Details(),
	}

	response := Response{
		StatusCode: statusCode,
		Message:    message,
		Error:      errInfo,
	}

	c.JSON(statusCode, response)
}

// Success sends a 200 OK response
func Success(c *gin.Context, message string, data interface{}) {
	Send(c, http.StatusOK, message, data)
}

// Created sends a 201 Created response
func Created(c *gin.Context, message string, data interface{}) {
	Send(c, http.StatusCreated, message, data)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
