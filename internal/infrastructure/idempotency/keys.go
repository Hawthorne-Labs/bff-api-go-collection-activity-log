package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const maxKeyLen = 200

// ValidateClientKey trims and validates Idempotency-Key.
// Returns key and reason "" | "required" | "invalid".
func ValidateClientKey(raw string) (string, string) {
	if raw == "" {
		return "", "required"
	}
	key := strings.TrimSpace(raw)
	if key == "" || len(key) > maxKeyLen {
		return "", "invalid"
	}
	return key, ""
}

// CanonicalHash is a stable SHA-256 over canonical JSON (sorted keys, no whitespace).
func CanonicalHash(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// ScopeKey binds idempotency to service + user + method + path.
func ScopeKey(service, user, method, path string) string {
	return fmt.Sprintf("%s:%s:%s:%s", service, user, method, path)
}
