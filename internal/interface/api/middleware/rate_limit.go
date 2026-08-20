package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RateLimitMiddleware(store RateLimitStore, trustedProxies string, skipPrefixes string) gin.HandlerFunc {
	trusted := parseTrustedProxies(trustedProxies)
	skips := parseSkipPrefixes(skipPrefixes)
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, prefix := range skips {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}
		clientIP := resolveClientIP(c.ClientIP(), c.GetHeader("X-Forwarded-For"), trusted)
		if store == nil || store.Allow(clientIP+":"+path) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": map[string]any{"code": 4290, "message": "Se excedió el límite de solicitudes."}})
	}
}

func parseTrustedProxies(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out[trimmed] = struct{}{}
		}
	}
	return out
}

func parseSkipPrefixes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func resolveClientIP(directHost, forwardedFor string, trusted map[string]struct{}) string {
	if !isTrustedProxy(directHost, trusted) || strings.TrimSpace(forwardedFor) == "" {
		return directHost
	}
	return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
}

func isTrustedProxy(host string, trusted map[string]struct{}) bool {
	if _, ok := trusted[host]; ok {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for proxy := range trusted {
		if _, network, err := net.ParseCIDR(proxy); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
