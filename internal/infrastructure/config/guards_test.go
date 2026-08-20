package config

import (
	"testing"
)

func TestEnforceProductionSecretsSkipsDevelopment(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("INTERNAL_JWT_SECRET", devDefaults["INTERNAL_JWT_SECRET"])
	if err := EnforceProductionSecrets(); err != nil {
		t.Fatalf("expected nil in development, got %v", err)
	}
}

func TestEnforceProductionSecretsRejectsDevDefaults(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("INTERNAL_JWT_SECRET", devDefaults["INTERNAL_JWT_SECRET"])
	t.Setenv("CORE_INTERNAL_TOKEN", devDefaults["CORE_INTERNAL_TOKEN"])
	t.Setenv("NOTIFICATION_CURSOR_SECRET", devDefaults["NOTIFICATION_CURSOR_SECRET"])
	t.Setenv("VAULT_TOKEN", devDefaults["VAULT_TOKEN"])
	t.Setenv("MTLS_BUNDLE_JSON", "bundle")
	t.Setenv("MTLS_EXPECTED_WORKLOAD_ID", "bff-activity-log")
	t.Setenv("MTLS_EXPECTED_USAGE", "client")
	t.Setenv("API_BASE_URL", "https://core.example")
	t.Setenv("CRYPTO_BFF_URL", "https://crypto.example")

	err := EnforceProductionSecrets()
	if err == nil {
		t.Fatal("expected insecure default error")
	}
}
