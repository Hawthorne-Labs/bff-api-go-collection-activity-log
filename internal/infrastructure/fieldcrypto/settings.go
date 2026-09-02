package fieldcrypto

import (
	"os"
	"strings"
)

const defaultMaxPayload = 256 * 1024

// CryptoSettings holds field-crypto configuration from environment.
type CryptoSettings struct {
	Enabled          bool
	FailClosed       bool
	MaxPayloadSize   int
	Mode             string
	Version          string
	LogCiphertext    bool
	LogPlaintext     bool
	Endpoints        [][4]any
}

func flagEnv(name string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if v == "" {
		return defaultVal
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseEndpoints(raw string) [][4]any {
	entries := make([][4]any, 0)
	for _, chunk := range strings.Split(raw, ";") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.Split(chunk, ":")
		if len(parts) != 4 {
			continue
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		entries = append(entries, [4]any{
			parts[0],
			parts[1],
			parts[2] == "decrypt",
			parts[3] == "encrypt",
		})
	}
	return entries
}

// CryptoSettingsFromEnv loads CRYPTO_* settings.
func CryptoSettingsFromEnv() *CryptoSettings {
	maxPayload := defaultMaxPayload
	if v := strings.TrimSpace(os.Getenv("CRYPTO_MAX_PAYLOAD_SIZE")); v != "" {
		n := 0
		for _, c := range v {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			maxPayload = n
		}
	}
	return &CryptoSettings{
		Enabled:        flagEnv("CRYPTO_ENABLED", false),
		FailClosed:     flagEnv("CRYPTO_FAIL_CLOSED", true),
		MaxPayloadSize: maxPayload,
		Mode:           getEnvDefault("CRYPTO_MODE", "field-values"),
		Version:        getEnvDefault("CRYPTO_VERSION", "v1"),
		LogCiphertext:  flagEnv("CRYPTO_LOG_CIPHERTEXT", false),
		LogPlaintext:   flagEnv("CRYPTO_LOG_PLAINTEXT", false),
		Endpoints:      parseEndpoints(os.Getenv("CRYPTO_REQUIRED_ENDPOINTS")),
	}
}

// Policy returns the effective crypto policy including defaults.
func (s *CryptoSettings) Policy() *CryptoPolicy {
	entries := [][4]any{
		{"PUT", "/api/v1/notifications/devices/current", true, false},
		// session-patch: require crypto session headers, no field decrypt/encrypt
		{"PATCH", "/api/v1/collections/escalations/{escalationId}/status", false, false},
		{"PATCH", "/api/v1/collections/escalations/{escalation_id}/status", false, false},
		{"PATCH", "/api/v1/collections/escalations/{id}/status", false, false},
		// anti-regresion: desempeno agent-performance reads must encrypt responses even if
		// CRYPTO_REQUIRED_ENDPOINTS env lags deploy (fixes CRYPTO_PLAINTEXT_REJECTED on Agentes tab).
		{"GET", "/api/v1/collections/agent-performance/ranking", false, true},
		{"GET", "/api/v1/collections/agent-performance/workload", false, true},
		{"GET", "/api/v1/collections/agent-performance/operations-summary", false, true},
		{"GET", "/api/v1/collections/agent-performance/goals", false, true},
		{"GET", "/api/v1/collections/agent-performance/report", false, true},
		{"GET", "/api/v1/collections/agent-performance/kpis", false, true},
	}
	entries = append(entries, s.Endpoints...)
	return FromEntries(entries)
}

func getEnvDefault(key, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultValue
}
