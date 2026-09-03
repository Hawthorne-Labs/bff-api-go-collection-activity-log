package activities

import "testing"

func TestCacheableList(t *testing.T) {
	t.Parallel()
	if !CacheableList("loan-1", nil, "", 0) {
		t.Fatal("loan_id first page")
	}
	if CacheableList("", []string{"a", "b"}, "", 0) {
		t.Fatal("loan_ids-only lists lack per-loan versioning and must not cache")
	}
	if !CacheableList("", nil, "client-1", 0) {
		t.Fatal("client_id first page")
	}
	if CacheableList("loan-1", nil, "", 20) {
		t.Fatal("offset>0 must not cache")
	}
	if CacheableList("", nil, "", 0) {
		t.Fatal("unscoped list must not cache")
	}
}

func TestMemoryReadCacheInvalidatesByLoanScope(t *testing.T) {
	t.Parallel()
	cache := NewMemoryReadCache()
	ctx := t.Context()
	payload := map[string]any{"items": []any{}, "total": 0}
	cache.SetList(ctx, "t1", "a@x.test", "loan-1", nil, "", "", "", "", 20, 0, payload)
	if _, ok := cache.GetList(ctx, "t1", "a@x.test", "loan-1", nil, "", "", "", "", 20, 0); !ok {
		t.Fatal("expected hit")
	}
	cache.InvalidateScopes(ctx, "t1", "loan-1", "")
	if _, ok := cache.GetList(ctx, "t1", "a@x.test", "loan-1", nil, "", "", "", "", 20, 0); ok {
		t.Fatal("expected miss after invalidate")
	}
}

func TestScopeIDsFromPayload(t *testing.T) {
	t.Parallel()
	loanID, clientID := ScopeIDsFromPayload(map[string]any{
		"loan_id":   " L1 ",
		"client_id": "C1",
	})
	if loanID != "L1" || clientID != "C1" {
		t.Fatalf("got %q %q", loanID, clientID)
	}
}
