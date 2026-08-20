package middleware

import "github.com/gin-gonic/gin"

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "0")
		c.Header("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'")
	}
}
