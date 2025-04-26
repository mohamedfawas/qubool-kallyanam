package errors

import (
	"errors"
	"fmt"
)

// AppError represents a simplified application error
type AppError struct {
	code    Code
	message string
	details map[string]interface{}
	err     error // Standard error for wrapping
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

// Unwrap implements the unwrap interface for standard error handling
func (e *AppError) Unwrap() error {
	return e.err
}

// Code returns the error code
func (e *AppError) Code() Code {
	return e.code
}

// Details returns error metadata
func (e *AppError) Details() map[string]interface{} {
	return e.details
}

// New creates a simple error with code and message
func New(code Code, message string) *AppError {
	return &AppError{
		code:    code,
		message: message,
		details: make(map[string]interface{}),
	}
}

// NewWithDetails creates an error with details included
func NewWithDetails(code Code, message string, details map[string]interface{}) *AppError {
	return &AppError{
		code:    code,
		message: message,
		details: details,
	}
}

// Wrap wraps an existing error
func Wrap(err error, code Code, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		code:    code,
		message: message,
		details: make(map[string]interface{}),
		err:     err,
	}
}

// WrapWithDetails wraps an error and adds details
func WrapWithDetails(err error, code Code, message string, details map[string]interface{}) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		code:    code,
		message: message,
		details: details,
		err:     err,
	}
}

// Domain-specific factory functions
func NotFound(message string) *AppError {
	return New(CodeNotFound, message)
}

func BadRequest(message string) *AppError {
	return New(CodeInvalidArgument, message)
}

func Unauthorized(message string) *AppError {
	return New(CodeUnauthenticated, message)
}

func Forbidden(message string) *AppError {
	return New(CodePermissionDenied, message)
}

func Internal(message string) *AppError {
	return New(CodeInternal, message)
}

// IsCode checks if error has a specific code
func IsCode(err error, code Code) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.code == code
}

// FromError converts any error to an AppError
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
		err:     err,
	}
}
