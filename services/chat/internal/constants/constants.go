package constants

// Collection names for MongoDB
const (
	ConversationsCollection = "conversations"
	MessagesCollection      = "messages"
)

// Message constraints
const (
	MaxMessageLength    = 2000
	DefaultMessageLimit = 20
	MaxMessageLimit     = 50
	MinMessageLimit     = 1
)

// Conversation constraints
const (
	DefaultConversationLimit = 20
	MaxConversationLimit     = 50
	MinParticipants          = 2
	MaxParticipants          = 2 // For direct messaging
)

// Event types for message broker
const (
	EventTypeMessageSent         = "message.sent"
	EventTypeConversationCreated = "conversation.created"
)

// Topics for message broker
const (
	TopicMessageSent         = "chat.message.sent"
	TopicConversationCreated = "chat.conversation.created"
)

// gRPC headers (for internal service communication)
const (
	AuthorizationHeader = "authorization"
	UserIDHeader        = "user-id"
	BearerPrefix        = "Bearer "
)
