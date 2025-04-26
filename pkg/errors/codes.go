package errors

// Code represents an error code
type Code string

// Application error codes
const (
	// Standard error codes
	CodeUnknown          Code = "UNKNOWN"           // Unknown or unspecified error
	CodeInternal         Code = "INTERNAL"          // Internal server error
	CodeInvalidArgument  Code = "INVALID_ARGUMENT"  // Invalid input data
	CodeNotFound         Code = "NOT_FOUND"         // Resource not found
	CodeAlreadyExists    Code = "ALREADY_EXISTS"    // Resource already exists
	CodePermissionDenied Code = "PERMISSION_DENIED" // Insufficient permissions
	CodeUnauthenticated  Code = "UNAUTHENTICATED"   // Authentication required

	// Domain-specific error codes
	CodeProfileIncomplete Code = "PROFILE_INCOMPLETE" // User profile is incomplete
	CodeInvalidMatch      Code = "INVALID_MATCH"      // Invalid matrimony match
	CodePaymentRequired   Code = "PAYMENT_REQUIRED"   // Payment is required
)

// codeInfo stores metadata about each error code
type codeInfo struct {
	HTTPStatus int // HTTP status code
	GRPCStatus int // gRPC status code
}

// codeMap maps error codes to their metadata
var codeMap = map[Code]codeInfo{
	CodeUnknown:          {500, 2},
	CodeInternal:         {500, 13},
	CodeInvalidArgument:  {400, 3},
	CodeNotFound:         {404, 5},
	CodeAlreadyExists:    {409, 6},
	CodePermissionDenied: {403, 7},
	CodeUnauthenticated:  {401, 16},

	// Domain-specific mappings
	CodeProfileIncomplete: {400, 9},
	CodeInvalidMatch:      {400, 3},
	CodePaymentRequired:   {402, 9},
}

// HTTPStatusCode returns the corresponding HTTP status code
func (c Code) HTTPStatusCode() int {
	if info, ok := codeMap[c]; ok {
		return info.HTTPStatus
	}
	return 500 // Default to Internal Server Error
}

// ToGRPCCode returns the corresponding gRPC status code
func (c Code) ToGRPCCode() int {
	if info, ok := codeMap[c]; ok {
		return info.GRPCStatus
	}
	return 2 // Default to Unknown
}
