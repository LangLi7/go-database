package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets HTTP security headers to protect against common attacks
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP - allow inline styles and fonts for the UI
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws://localhost:* http://localhost:*; "+
				"script-src 'self' 'unsafe-inline'; "+
				"frame-ancestors 'none'")

		c.Next()
	}
}
