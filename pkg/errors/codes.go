package errors

// Code represents an error code type
type Code string

// Common error codes shared across the application
const (
	// Standard error types
	CodeUnknown            Code = "UNKNOWN"
	CodeInternal           Code = "INTERNAL"
	CodeInvalidArgument    Code = "INVALID_ARGUMENT"
	CodeNotFound           Code = "NOT_FOUND"
	CodeAlreadyExists      Code = "ALREADY_EXISTS"
	CodePermissionDenied   Code = "PERMISSION_DENIED"
	CodeUnauthenticated    Code = "UNAUTHENTICATED"
	CodeResourceExhausted  Code = "RESOURCE_EXHAUSTED"
	CodeFailedPrecondition Code = "FAILED_PRECONDITION"
	CodeTimeout            Code = "TIMEOUT"
	CodeCanceled           Code = "CANCELED"

	// Domain-specific error codes for matrimony platform
	CodeProfileIncomplete Code = "PROFILE_INCOMPLETE"
	CodeInvalidMatch      Code = "INVALID_MATCH"
	CodePaymentRequired   Code = "PAYMENT_REQUIRED"
	CodeMessageBlocked    Code = "MESSAGE_BLOCKED"
	CodeProfileHidden     Code = "PROFILE_HIDDEN"
)

// ToGRPCCode converts our application error code to the corresponding gRPC status code
func (c Code) ToGRPCCode() int {
	switch c {
	case CodeUnknown:
		return 2 // Unknown
	case CodeInternal:
		return 13 // Internal
	case CodeInvalidArgument:
		return 3 // InvalidArgument
	case CodeNotFound:
		return 5 // NotFound
	case CodeAlreadyExists:
		return 6 // AlreadyExists
	case CodePermissionDenied:
		return 7 // PermissionDenied
	case CodeUnauthenticated:
		return 16 // Unauthenticated
	case CodeResourceExhausted:
		return 8 // ResourceExhausted
	case CodeFailedPrecondition:
		return 9 // FailedPrecondition
	case CodeTimeout:
		return 4 // DeadlineExceeded
	case CodeCanceled:
		return 1 // Canceled
	// Domain-specific codes mapped to appropriate gRPC codes
	case CodeProfileIncomplete:
		return 9 // FailedPrecondition
	case CodeInvalidMatch:
		return 3 // InvalidArgument
	case CodePaymentRequired:
		return 9 // FailedPrecondition
	case CodeMessageBlocked:
		return 7 // PermissionDenied
	case CodeProfileHidden:
		return 7 // PermissionDenied
	default:
		return 2 // Unknown
	}
}

// HTTPStatusCode converts our application error code to the corresponding HTTP status code
func (c Code) HTTPStatusCode() int {
	switch c {
	case CodeUnknown:
		return 500 // Internal Server Error
	case CodeInternal:
		return 500 // Internal Server Error
	case CodeInvalidArgument:
		return 400 // Bad Request
	case CodeNotFound:
		return 404 // Not Found
	case CodeAlreadyExists:
		return 409 // Conflict
	case CodePermissionDenied:
		return 403 // Forbidden
	case CodeUnauthenticated:
		return 401 // Unauthorized
	case CodeResourceExhausted:
		return 429 // Too Many Requests
	case CodeFailedPrecondition:
		return 400 // Bad Request
	case CodeTimeout:
		return 504 // Gateway Timeout
	case CodeCanceled:
		return 499 // Client Closed Request
	// Domain-specific codes mapped to appropriate HTTP codes
	case CodeProfileIncomplete:
		return 400 // Bad Request
	case CodeInvalidMatch:
		return 400 // Bad Request
	case CodePaymentRequired:
		return 402 // Payment Required
	case CodeMessageBlocked:
		return 403 // Forbidden
	case CodeProfileHidden:
		return 403 // Forbidden
	default:
		return 500 // Internal Server Error
	}
}
