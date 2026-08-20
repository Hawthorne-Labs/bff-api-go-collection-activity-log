package fieldcrypto

import "testing"

func TestPolicyResolvesMethodAndPathPatterns(t *testing.T) {
	policy := FromEntries([][4]any{
		{"POST", "/api/v1/collections/clients/search", true, true},
		{"GET", "/api/v1/collections/loans/{id}", false, true},
	})
	r := policy.Resolve("POST", "/api/v1/collections/clients/search")
	if r == nil || !r.DecryptRequest || !r.EncryptResponse {
		t.Fatal("expected POST search rule")
	}
	g := policy.Resolve("GET", "/api/v1/collections/loans/abc-123")
	if g == nil || g.DecryptRequest || !g.EncryptResponse {
		t.Fatal("expected GET loan rule")
	}
	if policy.Resolve("GET", "/api/v1/collections/clients/search") != nil {
		t.Fatal("expected nil")
	}
	if policy.Resolve("GET", "/api/v1/collections/loans/abc/extra") != nil {
		t.Fatal("expected nil for extra segment")
	}
}

func TestDefaultPolicyIncludesNotificationDevicePut(t *testing.T) {
	settings := &CryptoSettings{Enabled: true}
	rule := settings.Policy().Resolve("PUT", "/api/v1/notifications/devices/current")
	if rule == nil || !rule.DecryptRequest || rule.EncryptResponse {
		t.Fatal("expected decrypt-only default notification rule")
	}
}
