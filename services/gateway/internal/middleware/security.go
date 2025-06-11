package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers for payment pages
func SecurityHeaders() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Content Security Policy for Razorpay
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://checkout.razorpay.com https://api.razorpay.com; " +
			"style-src 'self' 'unsafe-inline' https://checkout.razorpay.com https://fonts.googleapis.com; " +
			"img-src 'self' data: https: https://checkout.razorpay.com https://api.razorpay.com; " +
			"connect-src 'self' https://api.razorpay.com https://checkout.razorpay.com https://lumberjack.razorpay.com; " +
			"frame-src 'self' https://api.razorpay.com https://checkout.razorpay.com; " +
			"font-src 'self' data: https: https://fonts.googleapis.com https://fonts.gstatic.com; " +
			"object-src 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self';"

		c.Header("Content-Security-Policy", csp)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Don't set HSTS for localhost/development
		if c.Request.Host != "localhost" && c.Request.Host != "127.0.0.1" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	})
}
