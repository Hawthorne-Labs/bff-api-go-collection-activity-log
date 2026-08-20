package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/session"
)

var csrfExemptPaths = map[string]struct{}{
	"/api/v1/auth/login":     {},
	"/api/v1/auth/callback":  {},
	"/api/v1/auth/dev-login": {},
}

var csrfMutatingMethods = map[string]struct{}{
	http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {}, http.MethodDelete: {},
}

func CSRFMiddleware(store *session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := csrfMutatingMethods[c.Request.Method]; !ok {
			c.Next()
			return
		}
		if _, exempt := csrfExemptPaths[c.Request.URL.Path]; exempt {
			c.Next()
			return
		}
		sessionID, err := c.Cookie(session.CookieName())
		if err != nil || sessionID == "" || store == nil {
			c.Next()
			return
		}
		payload := store.Get(sessionID)
		if payload == nil {
			c.Next()
			return
		}
		sent := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		expected := strings.TrimSpace(payload.CSRFToken)
		if expected == "" || sent == "" || !session.ConstantTimeEqual(sent, expected) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": map[string]any{"code": 4031, "message": "Token CSRF inválido o ausente."}})
			return
		}
		c.Next()
	}
}
