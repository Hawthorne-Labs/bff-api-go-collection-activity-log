package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/session"
)

const sessionContextKey = "bff_session_payload"

// RequireSession ensures a valid bff_session cookie exists and loads SessionPayload into context.
func RequireSession(store *session.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "no session"})
			return
		}
		sessionID, err := c.Cookie(session.CookieName())
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "no session"})
			return
		}
		payload := store.Get(sessionID)
		if payload == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "session expired"})
			return
		}
		c.Set(sessionContextKey, payload)
		c.Next()
	}
}

// SessionPayload returns the authenticated cookie session from context.
func SessionPayload(c *gin.Context) *session.SessionPayload {
	raw, ok := c.Get(sessionContextKey)
	if !ok {
		return nil
	}
	payload, ok := raw.(*session.SessionPayload)
	if !ok || payload == nil {
		return nil
	}
	return payload
}
