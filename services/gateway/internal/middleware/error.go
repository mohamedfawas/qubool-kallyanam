package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// ErrorHandler creates middleware for centralized error handling
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if any errors were recorded during the request and if response hasn't been sent yet.
		if len(c.Errors) > 0 && !c.Writer.Written() {
			// c.Errors is a list of all errors collected during the request.
			// c.Writer.Written() checks whether a response was already sent to the client.

			// Get the last error that occurred during this request.
			err := c.Errors.Last().Err // it will be stored here and retrieved as `err`.

			// Send the error to the client.
			http.Error(c, err)
		}
	}
}
