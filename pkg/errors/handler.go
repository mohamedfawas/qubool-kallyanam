package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Handler is an error handler that can process and format errors
type Handler struct {
	debug bool
}

// NewHandler creates a new error handler
func NewHandler(debug bool) *Handler {
	return &Handler{
		debug: debug,
	}
}

// FormatError formats an error into a structured error response
func (h *Handler) FormatError(err error) *ErrorResponse {
	appErr := FromError(err)
	if appErr == nil {
		return nil
	}

	details := make(map[string]interface{})
	for k, v := range appErr.details {
		details[k] = v
	}

	// In debug mode, add stack trace or cause information
	if h.debug && appErr.cause != nil {
		details["cause"] = appErr.cause.Error()
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
	statusCode := appErr.HTTPStatusCode()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If we can't encode the error, fall back to a simple error message
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
		logEntry[fmt.Sprintf("error_detail_%s", k)] = v
	}

	// Always log the cause in logs
	if appErr.cause != nil {
		logEntry["error_cause"] = appErr.cause.Error()
	}

	return logEntry
}
