package helpers

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/constants"
	chaterrors "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/errors"
)

// ValidateCreateConversationInput validates conversation creation input
func ValidateCreateConversationInput(userID, participantID string) error {
	if userID == "" || participantID == "" {
		return chaterrors.ErrInvalidInput
	}
	if userID == participantID {
		return chaterrors.ErrDuplicateParticipant
	}
	return nil
}

// ValidateSendMessageInput validates message sending input
func ValidateSendMessageInput(userID, conversationID, text string) error {
	if userID == "" || conversationID == "" || text == "" {
		return chaterrors.ErrInvalidInput
	}
	if len(text) > constants.MaxMessageLength {
		return chaterrors.ErrMessageTooLong
	}
	return nil
}

// ValidateObjectID validates and parses MongoDB ObjectID
func ValidateObjectID(id string) (primitive.ObjectID, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, chaterrors.ErrInvalidInput
	}
	return objectID, nil
}

// ValidatePaginationParams validates and normalizes pagination parameters
func ValidatePaginationParams(limit, offset int, maxLimit, defaultLimit int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
