package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// ErrorHandler creates middleware for centralized error handling
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// If there are errors and response hasn't been written
		if len(c.Errors) > 0 && !c.Writer.Written() {
			err := c.Errors.Last().Err
			http.Error(c, err)
		}
	}
}
