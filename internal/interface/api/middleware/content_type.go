package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var methodsWithBody = map[string]struct{}{
	http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {},
}

// RequireJSONContentType rejects POST/PUT/PATCH requests with a non-JSON body content-type.
func RequireJSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := methodsWithBody[c.Request.Method]; !ok {
			c.Next()
			return
		}
		if !hasBody(c.Request) {
			c.Next()
			return
		}
		contentType := strings.ToLower(strings.TrimSpace(strings.Split(c.GetHeader("Content-Type"), ";")[0]))
		if contentType != "application/json" {
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{
				"error": map[string]any{
					"code":    4150,
					"message": "content-type must be application/json",
				},
			})
			return
		}
		c.Next()
	}
}

func hasBody(r *http.Request) bool {
	raw := r.Header.Get("Content-Length")
	if raw == "" {
		return strings.EqualFold(r.Header.Get("Transfer-Encoding"), "chunked")
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return true
	}
	return n > 0
}
