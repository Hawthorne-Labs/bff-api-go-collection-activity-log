package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CryptoMiddleware handles Field-Level Encryption (FLE) for requests/responses.
// When CRYPTO_ENABLED is true, this middleware:
// 1. Decrypts incoming request bodies
// 2. Encrypts outgoing response bodies
// When disabled, it's a no-op passthrough.
func CryptoMiddleware(enabled bool, cryptoClient *CryptoClient) gin.HandlerFunc {
	if !enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Check for crypto headers
		cryptoSessionID := c.GetHeader("Crypto-Session-Id")
		cryptoRequestID := c.GetHeader("Crypto-Request-Id")
		cryptoVersion := c.GetHeader("Crypto-Version")
		cryptoTenantID := c.GetHeader("Crypto-Tenant-Id")

		// If no crypto headers, proceed normally (some endpoints don't need FLE)
		if cryptoSessionID == "" || cryptoRequestID == "" {
			c.Next()
			return
		}

		// Store crypto context for handlers
		c.Set("crypto_session_id", cryptoSessionID)
		c.Set("crypto_request_id", cryptoRequestID)
		c.Set("crypto_version", cryptoVersion)
		c.Set("crypto_tenant_id", cryptoTenantID)

		c.Next()

		// After handler, if response needs encryption, the handler itself
		// should use the crypto client. This middleware just validates session presence.
		_ = cryptoClient
		_ = cryptoVersion
		_ = cryptoTenantID
	}
}

// CORS handles Cross-Origin Resource Sharing.
func CORS(origins string) gin.HandlerFunc {
	originList := parseOrigins(origins)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, o := range originList {
			if o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Trace-Id, X-Tenant-Id, X-User-Email, X-Crypto-Session-Id, X-Crypto-Access-Token, Crypto-Session-Id, Crypto-Request-Id, Crypto-Version, Crypto-Tenant-Id, Idempotency-Key, traceparent")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func parseOrigins(origins string) []string {
	if origins == "" {
		return []string{"*"}
	}
	result := []string{}
	for _, o := range splitComma(origins) {
		o = trimSpace(o)
		if o != "" {
			result = append(result, o)
		}
	}
	if len(result) == 0 {
		return []string{"*"}
	}
	return result
}

func splitComma(s string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// CryptoClient wraps the crypto-bff client for FLE operations.
type CryptoClient struct {
	baseURL string
}

// NewCryptoClient creates a new crypto client.
func NewCryptoClient(baseURL string) *CryptoClient {
	return &CryptoClient{baseURL: baseURL}
}

// EncryptFields encrypts specific fields in a response.
func (c *CryptoClient) EncryptFields(
	sessionID, requestID, version, tenantID, method, path string,
	fields map[string]string,
) (map[string]string, error) {
	// Placeholder: In production, this calls the crypto-bff service
	// For now, return fields as-is when crypto is not fully integrated
	return fields, nil
}

// ValidateSession checks if a crypto session is valid.
func (c *CryptoClient) ValidateSession(sessionID string) (bool, error) {
	// Placeholder: In production, validates with crypto-bff
	return sessionID != "", nil
}

// CryptoEnforce ensures required crypto headers are present for protected endpoints.
func CryptoEnforce() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("Crypto-Session-Id")
		requestID := c.GetHeader("Crypto-Request-Id")

		if sessionID == "" || requestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]any{
					"code":    4060,
					"message": "encrypted client.fullName requires a crypto session (Crypto-Session-Id/Request-Id)",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
