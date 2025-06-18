package errors

import "errors"

// Common errors
var (
	ErrInvalidInput        = errors.New("invalid input parameters")
	ErrInternalServerError = errors.New("internal server error")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrForbidden           = errors.New("forbidden access")
)
