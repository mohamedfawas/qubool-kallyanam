package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSOptions holds CORS configuration options
type CORSOptions struct {
	// AllowAllOrigins allows all origins
	AllowAllOrigins bool

	// AllowOrigins is a list of allowed origins
	AllowOrigins []string

	// AllowMethods is a list of allowed HTTP methods
	AllowMethods []string

	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool

	// MaxAge indicates how long preflight results can be cached
	MaxAge int
}

// DefaultCORSOptions returns sensible defaults
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowAllOrigins:  false,
		AllowOrigins:     []string{"http://localhost:*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles CORS
func CORS(options ...CORSOptions) gin.HandlerFunc {
	// Use default options or provided options
	opts := DefaultCORSOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	// Create CORS config
	config := cors.DefaultConfig()

	if opts.AllowAllOrigins {
		config.AllowAllOrigins = true
	} else {
		config.AllowOrigins = opts.AllowOrigins
	}

	config.AllowMethods = opts.AllowMethods
	config.AllowCredentials = opts.AllowCredentials
	config.MaxAge = time.Duration(opts.MaxAge) * time.Second

	return cors.New(config)
}
