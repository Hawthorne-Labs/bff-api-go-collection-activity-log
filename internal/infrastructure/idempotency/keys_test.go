package idempotency

import "testing"

func TestValidateClientKey(t *testing.T) {
	if _, reason := ValidateClientKey(""); reason != "required" {
		t.Fatalf("expected required, got %q", reason)
	}
	if _, reason := ValidateClientKey("   "); reason != "invalid" {
		t.Fatalf("expected invalid, got %q", reason)
	}
	key, reason := ValidateClientKey("batch-key-12345678")
	if reason != "" || key != "batch-key-12345678" {
		t.Fatalf("unexpected key=%q reason=%q", key, reason)
	}
}

func TestCanonicalHashStable(t *testing.T) {
	a, err := CanonicalHash(map[string]any{"b": 1, "a": 2})
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalHash(map[string]any{"a": 2, "b": 1})
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("hash must be key-order stable: %s vs %s", a, b)
	}
}
