package constants

import "time"

// Payment related constants
const (
	// Currency
	DefaultCurrency = "INR"

	// Amount conversion
	PaiseMultiplier = 100

	// Default pagination
	DefaultLimit  = 10
	MaxLimit      = 100
	DefaultOffset = 0

	// Subscription duration
	DefaultPlanDurationYears = 1
	DefaultPlanDurationDays  = 365

	// Default plan configuration
	DefaultPlanID     = "premium_365"
	DefaultPlanName   = "Premium Membership"
	DefaultPlanAmount = 1000.0

	// Timeouts and retries
	DefaultTimeout = 30 * time.Second
)

// Status constants
const (
	StatusActive    = "active"
	StatusPending   = "pending"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

// Error messages
const (
	ErrMsgInvalidPlan                 = "invalid subscription plan"
	ErrMsgPaymentNotFound             = "payment not found"
	ErrMsgInvalidSignature            = "invalid payment signature"
	ErrMsgDuplicatePayment            = "payment already processed"
	ErrMsgOrderCreationFailed         = "failed to create payment order"
	ErrMsgPaymentSaveFailed           = "failed to save payment record"
	ErrMsgSignatureVerificationFailed = "payment signature verification failed"
)
