package errors

import "errors"

// Profile service errors
var (
	ErrInvalidInput    = errors.New("invalid input parameters")
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileExists   = errors.New("profile already exists")
	ErrValidation      = errors.New("validation error")
)

// Partner preferences errors
var (
	ErrInvalidAgeRange            = errors.New("invalid age range: min age must be at least 18, max age must be at most 80, and min age must not exceed max age")
	ErrInvalidHeightRange         = errors.New("invalid height range: min height must be at least 130cm, max height must be at most 220cm, and min height must not exceed max height")
	ErrPartnerPreferencesNotFound = errors.New("partner preferences not found")
)

// Matchmaking service errors
var (
	ErrMatchActionFailed       = errors.New("failed to record match action")
	ErrNoMatches               = errors.New("no matches found")
	ErrInvalidActionType       = errors.New("invalid action type")
	ErrInvalidMatchingCriteria = errors.New("invalid matching criteria")
	ErrMatchNotFound           = errors.New("match not found")
)

// File and media errors
var (
	ErrInvalidFileType      = errors.New("invalid file type")
	ErrFileTooLarge         = errors.New("file size exceeds maximum allowed size")
	ErrFileProcessingFailed = errors.New("failed to process file")
)

// Photo service errors
var (
	ErrPhotoUploadFailed = errors.New("failed to upload photo")
	ErrPhotoDeleteFailed = errors.New("failed to delete photo")
	ErrPhotoNotFound     = errors.New("photo not found")
	ErrPhotoLimitReached = errors.New("maximum photo limit reached")
)

// Video service errors
var (
	ErrVideoUploadFailed = errors.New("failed to upload video")
	ErrVideoDeleteFailed = errors.New("failed to delete video")
	ErrVideoNotFound     = errors.New("video not found")
	ErrVideoTooLong      = errors.New("video duration exceeds maximum allowed length")
)

// Pagination errors
var (
	ErrInvalidPaginationLimit  = errors.New("invalid pagination limit: must be between 1 and 100")
	ErrInvalidPaginationOffset = errors.New("invalid pagination offset: must be greater than or equal to 0")
)

// Authentication and authorization errors
var (
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrInvalidToken      = errors.New("invalid authentication token")
	ErrTokenExpired      = errors.New("authentication token has expired")
	ErrInsufficientRoles = errors.New("insufficient permissions for this operation")
)

// Database and storage errors
var (
	ErrDatabaseConnection = errors.New("database connection failed")
	ErrStorageConnection  = errors.New("storage service connection failed")
	ErrDataIntegrity      = errors.New("data integrity violation")
)

// Business logic errors
var (
	ErrBusinessRule     = errors.New("business rule violation")
	ErrStateConflict    = errors.New("operation conflicts with current state")
	ErrResourceConflict = errors.New("resource conflict detected")
)
