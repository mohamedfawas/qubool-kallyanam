package errors

import "errors"

// Common errors
var (
	ErrInvalidInput        = errors.New("invalid input parameters")
	ErrInternalServerError = errors.New("internal server error")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrForbidden           = errors.New("forbidden access")
)

// Conversation errors
var (
	ErrConversationNotFound      = errors.New("conversation not found")
	ErrConversationAlreadyExists = errors.New("conversation already exists")
	ErrDuplicateParticipant      = errors.New("cannot create conversation with yourself")
	ErrInvalidParticipants       = errors.New("invalid participants")
	ErrMaxParticipantsExceeded   = errors.New("maximum participants exceeded")
)

// Message errors
var (
	ErrMessageNotFound     = errors.New("message not found")
	ErrMessageTooLong      = errors.New("message text exceeds maximum length")
	ErrEmptyMessage        = errors.New("message text cannot be empty")
	ErrMessageDeleted      = errors.New("message has been deleted")
	ErrCannotDeleteMessage = errors.New("cannot delete message")
)

// Authentication errors
var (
	ErrUserNotParticipant = errors.New("user is not a participant in this conversation")
	ErrInvalidUserID      = errors.New("invalid user ID")
)
