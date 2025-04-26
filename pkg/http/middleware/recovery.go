package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/response"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Recovery returns a middleware that recovers from panics
func Recovery(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic
				logger.Error("Panic recovered",
					logging.Any("error", r),
					logging.String("stack", stack),
				)

				// Create a server error
				err := errors.Internal("Internal server error")

				// Send error response
				response.SendError(c, err, "An unexpected error occurred")

				// Abort the request
				c.Abort()
			}
		}()

		c.Next()
	}
}
