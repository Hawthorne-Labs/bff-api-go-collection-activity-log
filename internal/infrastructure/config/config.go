package config

import (
	"net/http"
	"os"
	"strings"
)

// Config holds all environment-based configuration for the activity-log BFF.
type Config struct {
	Port                         string
	CoreBaseURL                  string
	CryptoBFFBaseURL             string
	CryptoEnabled                bool
	RequestTimeoutSeconds        int
	MaxRequestBodyBytes          int
	RateLimitRequests            int
	RateLimitWindowSec           int
	AWSRegion                    string
	CognitoPoolID                string
	CognitoIssuer                string
	CognitoAudience              string
	CognitoJWKSURL               string
	TrustedProxies               string
	RateLimitSkipPaths           string
	CORSSOrigins                 string
	SessionBackend               string
	LogLevel                     string
	OTELServiceName              string
	CryptoSessionSecret          string
	CryptoSessionIssuer          string
	CryptoSessionTTL             int
	InternalJWTSecret            string
	InternalJWTIssuer            string
	InternalJWTCoreAudience      string
	InternalJWTActiveKID         string
	InternalJWTCryptoAudience    string
	CryptoVersion                string
	CryptoMode                   string
	CryptoFailClosed             bool
	CryptoMaxPayloadSize         int
	CryptoLogCiphertext          bool
	CryptoLogPlaintext           bool
	CryptoRequiredEndpoints      string
	CryptoKeys                   string
	CryptoActiveKID              string
	CryptoSessionMode            string
	CryptoSessionBackend         string
	CryptoSessionNamespace       string
	CryptoSessionRedisRollback   bool
	CryptoSessionKEKActiveKID    string
	CryptoSessionKEKActiveB64   string
	CryptoSessionKEKPreviousKID  string
	CryptoSessionKEKPreviousB64 string
	CryptoTenantDigestKey        string
	ManagementCoreBaseURL        string
	IdentityCoreBaseURL          string
	ManagementCoreTimeoutSec     float64
	NotificationCursorSecret     string
	NotificationStreamMaxSeconds int
	RedisURL                     string
	Environment                  string
	KeycloakURL                  string
	KeycloakPublicURL            string
	KeycloakRealm                string
	KeycloakClientID             string
	KeycloakTimeoutSeconds       int
	BFFOIDCRedirectURI           string
	BFFPostLoginURL              string
	BFFPostLogoutURL             string
	BFFCookieSecure              bool
	BFFCookieSameSite            string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:                         getEnvOrDefault("PORT", "8000"),
		CoreBaseURL:                  getEnvOrDefault("CORE_BASE_URL", getEnvOrDefault("API_BASE_URL", "http://localhost:9090")),
		CryptoBFFBaseURL:             getEnvOrDefault("CRYPTO_BFF_BASE_URL", getEnvOrDefault("CRYPTO_BFF_URL", "http://localhost:8081")),
		CryptoEnabled:                isTrueEnv("CRYPTO_ENABLED"),
		RequestTimeoutSeconds:        getEnvIntOrDefault("REQUEST_TIMEOUT_SECONDS", 30),
		MaxRequestBodyBytes:          getEnvIntOrDefault("MAX_REQUEST_BODY_BYTES", 65536),
		RateLimitRequests:            getEnvIntOrDefault("RATE_LIMIT_REQUESTS", 60),
		RateLimitWindowSec:           getEnvIntOrDefault("RATE_LIMIT_WINDOW_SECONDS", 60),
		AWSRegion:                    getEnvOrDefault("AWS_REGION", "us-east-1"),
		CognitoPoolID:                getEnvOrDefault("COGNITO_POOL_ID", ""),
		CognitoIssuer:                getEnvOrDefault("COGNITO_ISSUER", ""),
		CognitoAudience:              getEnvOrDefault("COGNITO_AUDIENCE", getEnvOrDefault("COGNITO_CLIENT_ID", "")),
		CognitoJWKSURL:               getEnvOrDefault("COGNITO_JWKS_URL", ""),
		TrustedProxies:               getEnvOrDefault("TRUSTED_PROXIES", "127.0.0.1"),
		RateLimitSkipPaths:           getEnvOrDefault("RATE_LIMIT_SKIP_PATHS", "/health"),
		CORSSOrigins:                 getEnvOrDefault("BFF_CORS_ORIGINS", "http://localhost:5173"),
		SessionBackend:               getEnvOrDefault("SESSION_BACKEND", "redis"),
		LogLevel:                     getEnvOrDefault("LOG_LEVEL", "info"),
		OTELServiceName:              getEnvOrDefault("OTEL_SERVICE_NAME", "bff-api-go-collection-activity-log"),
		CryptoSessionSecret:          getEnvOrDefault("CRYPTO_SESSION_TOKEN_SECRET", getEnvOrDefault("INTERNAL_JWT_SECRET", "dev-internal-jwt-secret-32-bytes-min")),
		CryptoSessionIssuer:          getEnvOrDefault("CRYPTO_SESSION_ISSUER", "hawthorne-bff"),
		CryptoSessionTTL:             getEnvIntOrDefault("CRYPTO_SESSION_TTL_SECONDS", 900),
		InternalJWTSecret:            getEnvOrDefault("INTERNAL_JWT_SECRET", getEnvOrDefault("CORE_JWT_SECRET", "dev-internal-jwt-secret-32-bytes-min")),
		InternalJWTIssuer:            getEnvOrDefault("INTERNAL_JWT_ISSUER", "python-templates-finch"),
		InternalJWTCoreAudience:      getEnvOrDefault("INTERNAL_JWT_CORE_AUDIENCE", "core-api"),
		InternalJWTActiveKID:         getEnvOrDefault("INTERNAL_JWT_ACTIVE_KID", ""),
		InternalJWTCryptoAudience:    getEnvOrDefault("INTERNAL_JWT_CRYPTO_AUDIENCE", "crypto-bff-decrypt"),
		CryptoVersion:                getEnvOrDefault("CRYPTO_VERSION", "v1"),
		CryptoMode:                   getEnvOrDefault("CRYPTO_MODE", "field-values"),
		CryptoFailClosed:             isTrueEnvDefault("CRYPTO_FAIL_CLOSED", true),
		CryptoMaxPayloadSize:         getEnvIntOrDefault("CRYPTO_MAX_PAYLOAD_SIZE", 256*1024),
		CryptoLogCiphertext:          isTrueEnv("CRYPTO_LOG_CIPHERTEXT"),
		CryptoLogPlaintext:           isTrueEnv("CRYPTO_LOG_PLAINTEXT"),
		CryptoRequiredEndpoints:      os.Getenv("CRYPTO_REQUIRED_ENDPOINTS"),
		CryptoKeys:                   os.Getenv("CRYPTO_KEYS"),
		CryptoActiveKID:              os.Getenv("CRYPTO_ACTIVE_KID"),
		CryptoSessionMode:            os.Getenv("CRYPTO_SESSION_MODE"),
		CryptoSessionBackend:         os.Getenv("CRYPTO_SESSION_BACKEND"),
		CryptoSessionNamespace:       getEnvOrDefault("CRYPTO_SESSION_NAMESPACE", "activity-log"),
		CryptoSessionRedisRollback:   isTrueEnv("CRYPTO_SESSION_REDIS_ROLLBACK"),
		CryptoSessionKEKActiveKID:    os.Getenv("CRYPTO_SESSION_KEK_ACTIVE_KID"),
		CryptoSessionKEKActiveB64:    os.Getenv("CRYPTO_SESSION_KEK_ACTIVE_B64"),
		CryptoSessionKEKPreviousKID:  os.Getenv("CRYPTO_SESSION_KEK_PREVIOUS_KID"),
		CryptoSessionKEKPreviousB64:  os.Getenv("CRYPTO_SESSION_KEK_PREVIOUS_B64"),
		CryptoTenantDigestKey:        os.Getenv("CRYPTO_TENANT_DIGEST_KEY"),
		ManagementCoreBaseURL:        os.Getenv("MANAGEMENT_CORE_API_BASE_URL"),
		IdentityCoreBaseURL:          os.Getenv("IDENTITY_CORE_API_BASE_URL"),
		ManagementCoreTimeoutSec:     getEnvFloatOrDefault("MANAGEMENT_CORE_TIMEOUT_SECONDS", 5.0),
		NotificationCursorSecret:     getEnvOrDefault("NOTIFICATION_CURSOR_SECRET", "dev-notification-cursor-secret"),
		NotificationStreamMaxSeconds: getEnvIntOrDefault("NOTIFICATION_STREAM_MAX_SECONDS", 300),
		RedisURL:                     getEnvOrDefault("REDIS_URL", ""),
		Environment:                  getEnvOrDefault("ENVIRONMENT", "development"),
		KeycloakURL:                  getEnvOrDefault("KEYCLOAK_URL", "http://keycloak:8080"),
		KeycloakPublicURL:            getEnvOrDefault("KEYCLOAK_PUBLIC_URL", ""),
		KeycloakRealm:                getEnvOrDefault("KEYCLOAK_REALM", "python-templates-finch"),
		KeycloakClientID:             getEnvOrDefault("KEYCLOAK_CLIENT_ID", "webapp"),
		KeycloakTimeoutSeconds:       getEnvIntOrDefault("KEYCLOAK_TIMEOUT_SECONDS", 5),
		BFFOIDCRedirectURI:           getEnvOrDefault("BFF_OIDC_REDIRECT_URI", "http://localhost:8080/api/v1/auth/callback"),
		BFFPostLoginURL:              getEnvOrDefault("BFF_POST_LOGIN_URL", "http://localhost:5173/"),
		BFFPostLogoutURL:             getEnvOrDefault("BFF_POST_LOGOUT_URL", "http://localhost:5173/"),
		BFFCookieSecure:              isTrueEnv("BFF_COOKIE_SECURE"),
		BFFCookieSameSite:            getEnvOrDefault("BFF_COOKIE_SAMESITE", "lax"),
	}
}

// IsDevLoginEnabled returns true when dev-login is allowed (non-production).
func (c *Config) IsDevLoginEnabled() bool {
	return strings.ToLower(strings.TrimSpace(c.Environment)) != "production"
}

// CookieSameSite maps BFF_COOKIE_SAMESITE to http.SameSite.
func (c *Config) CookieSameSite() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(c.BFFCookieSameSite)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n := 0
	for _, c := range v {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n <= 0 {
		return defaultVal
	}
	return n
}

func isTrueEnv(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func isTrueEnvDefault(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return isTrueEnv(key)
}

func getEnvFloatOrDefault(key string, defaultVal float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	n := 0.0
	dec := false
	div := 1.0
	for _, c := range v {
		if c == '.' {
			dec = true
			continue
		}
		if c >= '0' && c <= '9' {
			if dec {
				div *= 10
				n += float64(c-'0') / div
			} else {
				n = n*10 + float64(c-'0')
			}
		}
	}
	if n <= 0 {
		return defaultVal
	}
	return n
}
