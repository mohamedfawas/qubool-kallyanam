package errors

import "errors"

// Admin authentication errors
var (
	ErrAdminNotFound        = errors.New("admin not found")
	ErrAdminAccountDisabled = errors.New("admin account is disabled")
	ErrInvalidAdminInput    = errors.New("invalid admin input parameters")
)
