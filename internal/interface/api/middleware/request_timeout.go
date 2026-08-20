package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestTimeoutMiddleware(timeoutSeconds int) gin.HandlerFunc {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{"error": map[string]any{"code": 5040, "message": "La solicitud excedió el tiempo de espera."}})
		}
	}
}
