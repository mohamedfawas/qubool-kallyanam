package helpers

import (
	chaterrors "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapErrorToGRPCStatus maps internal errors to gRPC status codes
func MapErrorToGRPCStatus(err error) error {
	switch err {
	// Common errors
	case chaterrors.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, err.Error())
	case chaterrors.ErrUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case chaterrors.ErrForbidden:
		return status.Error(codes.PermissionDenied, err.Error())

	// Conversation errors
	case chaterrors.ErrConversationNotFound:
		return status.Error(codes.NotFound, err.Error())
	case chaterrors.ErrConversationAlreadyExists:
		return status.Error(codes.AlreadyExists, err.Error())
	case chaterrors.ErrDuplicateParticipant:
		return status.Error(codes.InvalidArgument, err.Error())
	case chaterrors.ErrInvalidParticipants:
		return status.Error(codes.InvalidArgument, err.Error())
	case chaterrors.ErrMaxParticipantsExceeded:
		return status.Error(codes.InvalidArgument, err.Error())

	// Message errors
	case chaterrors.ErrMessageNotFound:
		return status.Error(codes.NotFound, err.Error())
	case chaterrors.ErrMessageTooLong:
		return status.Error(codes.InvalidArgument, err.Error())
	case chaterrors.ErrEmptyMessage:
		return status.Error(codes.InvalidArgument, err.Error())
	case chaterrors.ErrMessageDeleted:
		return status.Error(codes.NotFound, "Message has been deleted")
	case chaterrors.ErrCannotDeleteMessage:
		return status.Error(codes.PermissionDenied, err.Error())

	// Authentication errors
	case chaterrors.ErrUserNotParticipant:
		return status.Error(codes.PermissionDenied, err.Error())
	case chaterrors.ErrInvalidUserID:
		return status.Error(codes.InvalidArgument, err.Error())

	// Default case
	default:
		return status.Error(codes.Internal, "Internal server error")
	}
}
