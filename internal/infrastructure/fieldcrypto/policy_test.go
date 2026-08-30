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

func TestDefaultPolicyEscalationStatusIsSessionPatch(t *testing.T) {
	settings := &CryptoSettings{Enabled: true}
	rule := settings.Policy().Resolve("PATCH", "/api/v1/collections/escalations/esc-1/status")
	if rule == nil {
		t.Fatal("expected session-patch rule for escalation status")
	}
	if rule.DecryptRequest || rule.EncryptResponse {
		t.Fatalf("session-patch must be none:none, got decrypt=%v encrypt=%v", rule.DecryptRequest, rule.EncryptResponse)
	}
}

func TestDefaultPolicyDoesNotCoverMedia(t *testing.T) {
	settings := &CryptoSettings{Enabled: true}
	policy := settings.Policy()
	if policy.Resolve("GET", "/api/v1/collections/clients/c1/media") != nil {
		t.Fatal("media must remain without activity-log FLE policy")
	}
}
