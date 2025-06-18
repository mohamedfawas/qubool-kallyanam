package constants

// Redis key prefixes
const (
	RefreshTokenPrefix = "refresh_token:"
	BlacklistPrefix    = "blacklist:"
	OTPPrefix          = "otp:"
)

// gRPC headers (for internal service communication)
const (
	AuthorizationHeader = "authorization"
	UserIDHeader        = "user-id"
	BearerPrefix        = "Bearer "
)

// Event types for message broker
const (
	EventTypeLogin       = "login"
	EventTypeUserDeleted = "user.deleted"
)

// Topics for message broker
const (
	TopicUserLogin             = "user.login"
	TopicUserDeleted           = "user.deleted"
	TopicSubscriptionActivated = "subscription.activated"
	TopicSubscriptionExtended  = "subscription.extended"
)

// Auth service specific constants
const (
	DefaultOTPLength     = 6
	DefaultPendingExpiry = 1 // hours
	MinPasswordLength    = 8
)
