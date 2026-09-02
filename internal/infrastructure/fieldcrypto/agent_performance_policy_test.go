package fieldcrypto

import "strings"
import "testing"

func TestActivityLogAgentPerformanceEndpointsRequireResponseEncryption(t *testing.T) {
	raw := strings.TrimSpace(getEnvDefault("CRYPTO_REQUIRED_ENDPOINTS", ""))
	if raw == "" {
		raw = "GET:/api/v1/collections/agent-performance/ranking:none:encrypt;GET:/api/v1/collections/agent-performance/workload:none:encrypt;GET:/api/v1/collections/agent-performance/operations-summary:none:encrypt;GET:/api/v1/collections/agent-performance/goals:none:encrypt;GET:/api/v1/collections/agent-performance/report:none:encrypt;GET:/api/v1/collections/agent-performance/kpis:none:encrypt"
	}
	settings := &CryptoSettings{
		Enabled:   true,
		Endpoints: parseEndpoints(raw),
	}
	policy := settings.Policy()
	for _, route := range []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/collections/agent-performance/workload"},
		{"GET", "/api/v1/collections/agent-performance/goals"},
		{"GET", "/api/v1/collections/agent-performance/report"},
		{"GET", "/api/v1/collections/agent-performance/kpis"},
		{"GET", "/api/v1/collections/agent-performance/ranking"},
		{"GET", "/api/v1/collections/agent-performance/operations-summary"},
	} {
		rule := policy.Resolve(route.method, route.path)
		if rule == nil {
			t.Fatalf("expected FLE policy for %s %s", route.method, route.path)
		}
		if rule.DecryptRequest {
			t.Fatalf("%s %s must not decrypt GET requests", route.method, route.path)
		}
		if !rule.EncryptResponse {
			t.Fatalf("%s %s must encrypt responses", route.method, route.path)
		}
	}
}
