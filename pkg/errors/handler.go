package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Code    string                 `json:"code"`              // Error code
	Message string                 `json:"message"`           // Error message
	Details map[string]interface{} `json:"details,omitempty"` // Additional error details
}

// Handler processes application errors
type Handler struct {
	debug bool
}

// NewHandler creates a new error handler
func NewHandler(debug bool) *Handler {
	return &Handler{
		debug: debug,
	}
}

// FormatError formats an error into a structured response
func (h *Handler) FormatError(err error) *ErrorResponse {
	appErr := FromError(err)
	if appErr == nil {
		return nil
	}

	details := make(map[string]interface{})
	for k, v := range appErr.details {
		details[k] = v
	}

	// In debug mode, add cause information
	if h.debug && appErr.err != nil {
		details["cause"] = appErr.err.Error()
	}

	return &ErrorResponse{
		Code:    string(appErr.code),
		Message: appErr.message,
		Details: details,
	}
}

// WriteHTTPError writes an error response to an HTTP response writer
func (h *Handler) WriteHTTPError(w http.ResponseWriter, err error) {
	appErr := FromError(err)
	if appErr == nil {
		return
	}

	response := h.FormatError(err)
	statusCode := appErr.code.HTTPStatusCode()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback if encoding fails
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"code":"INTERNAL","message":"Failed to encode error response"}`)
	}
}

// LogError formats an error for logging purposes
func (h *Handler) LogError(err error) map[string]interface{} {
	appErr := FromError(err)
	if appErr == nil {
		return nil
	}

	logEntry := map[string]interface{}{
		"error_code":    string(appErr.code),
		"error_message": appErr.message,
	}

	// Include error details in logs
	for k, v := range appErr.details {
		logEntry[fmt.Sprintf("detail_%s", k)] = v
	}

	// Always log the cause in logs
	if appErr.err != nil {
		logEntry["error_cause"] = appErr.err.Error()
	}

	return logEntry
}
