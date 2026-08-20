package config

import (
	"fmt"
	"os"
	"strings"
)

const devEnvironment = "development"

var devDefaults = map[string]string{
	"INTERNAL_JWT_SECRET":        "dev-internal-jwt-secret-32-bytes-min",
	"CORE_INTERNAL_TOKEN":        "dev-internal-jwt-secret-32-bytes-min",
	"NOTIFICATION_CURSOR_SECRET": "dev-notification-cursor-secret",
	"VAULT_TOKEN":                "root",
}

var productionMtlsIdentity = []string{
	"MTLS_BUNDLE_JSON",
	"MTLS_EXPECTED_WORKLOAD_ID",
	"MTLS_EXPECTED_USAGE",
}

var productionHTTPSURLs = []string{
	"API_BASE_URL",
	"CRYPTO_BFF_URL",
}

// InsecureDefaultError is returned when production starts with development defaults.
type InsecureDefaultError struct {
	Violations []string
}

func (e *InsecureDefaultError) Error() string {
	return fmt.Sprintf(
		"refusing to start: %s still use development defaults but ENVIRONMENT=%q",
		strings.Join(e.Violations, ", "),
		os.Getenv("ENVIRONMENT"),
	)
}

// EnforceProductionSecrets refuses startup in non-development when insecure defaults remain.
func EnforceProductionSecrets() error {
	env := os.Getenv("ENVIRONMENT")
	if strings.EqualFold(strings.TrimSpace(env), devEnvironment) || env == "" {
		return nil
	}

	violations := make([]string, 0, 8)
	for name, devValue := range devDefaults {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			if devValue != "" {
				violations = append(violations, name)
			}
			continue
		}
		if os.Getenv(name) == devValue {
			violations = append(violations, name)
		}
	}
	for _, name := range productionMtlsIdentity {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			violations = append(violations, name)
		}
	}
	for _, name := range productionHTTPSURLs {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
		if value == "" || !strings.HasPrefix(value, "https://") {
			violations = append(violations, name)
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return &InsecureDefaultError{Violations: violations}
}
