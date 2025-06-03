package errors

import (
	"errors"
	"fmt"
)

// Domain errors
var (
	ErrInvalidPlan          = errors.New("invalid subscription plan")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrInvalidSignature     = errors.New("invalid payment signature")
	ErrDuplicatePayment     = errors.New("payment already processed")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrUnauthorizedAccess   = errors.New("unauthorized access to payment")
)

// Service errors
var (
	ErrOrderCreation    = errors.New("failed to create payment order")
	ErrPaymentSave      = errors.New("failed to save payment record")
	ErrSignatureVerify  = errors.New("payment signature verification failed")
	ErrSubscriptionSave = errors.New("failed to save subscription")
)

// Error types for categorization
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "VALIDATION"
	ErrorTypeNotFound   ErrorType = "NOT_FOUND"
	ErrorTypeConflict   ErrorType = "CONFLICT"
	ErrorTypeInternal   ErrorType = "INTERNAL"
	ErrorTypeExternal   ErrorType = "EXTERNAL"
)

// PaymentError provides structured error information
type PaymentError struct {
	Type    ErrorType
	Code    string
	Message string
	Err     error
}

func (e *PaymentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Type, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

func (e *PaymentError) Unwrap() error {
	return e.Err
}

// Error constructors
func NewValidationError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Type:    ErrorTypeValidation,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewNotFoundError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Type:    ErrorTypeNotFound,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewConflictError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Type:    ErrorTypeConflict,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewInternalError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Type:    ErrorTypeInternal,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewExternalError(code, message string, err error) *PaymentError {
	return &PaymentError{
		Type:    ErrorTypeExternal,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes
const (
	CodeInvalidPlan      = "INVALID_PLAN"
	CodePaymentNotFound  = "PAYMENT_NOT_FOUND"
	CodeInvalidSignature = "INVALID_SIGNATURE"
	CodeDuplicatePayment = "DUPLICATE_PAYMENT"
	CodeOrderCreation    = "ORDER_CREATION_FAILED"
	CodePaymentSave      = "PAYMENT_SAVE_FAILED"
	CodeSignatureVerify  = "SIGNATURE_VERIFY_FAILED"
	CodeUnauthorized     = "UNAUTHORIZED_ACCESS"
)
