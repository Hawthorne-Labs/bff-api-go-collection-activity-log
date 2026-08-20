package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequestSizeLimitMiddleware(maxBodyBytes int) gin.HandlerFunc {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 65536
	}
	bodyMethods := map[string]struct{}{
		http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {},
	}
	return func(c *gin.Context) {
		if contentLength := strings.TrimSpace(c.GetHeader("Content-Length")); contentLength != "" {
			if size, err := strconv.Atoi(contentLength); err == nil && size > maxBodyBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": map[string]any{"code": 4130, "message": "El cuerpo de la solicitud es demasiado grande."}})
				return
			}
		}
		if _, ok := bodyMethods[c.Request.Method]; !ok {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(maxBodyBytes))
		c.Next()
	}
}
