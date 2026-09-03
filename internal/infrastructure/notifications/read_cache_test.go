package notifications

import "testing"

func TestCacheableListPage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                               string
		state, severity, from, to, at, id string
		want                               bool
	}{
		{name: "first unread page", state: "unread", want: true},
		{name: "first all page", state: "all", want: true},
		{name: "cursor page", state: "unread", at: "2026-01-01T00:00:00Z", id: "n1", want: false},
		{name: "date filter", state: "unread", from: "2026-01-01", want: false},
		{name: "unknown state", state: "archived", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CacheableListPage(tc.state, tc.severity, tc.from, tc.to, tc.at, tc.id)
			if got != tc.want {
				t.Fatalf("CacheableListPage()=%v want %v", got, tc.want)
			}
		})
	}
}

func TestMemoryReadCacheInvalidatesUnreadAndList(t *testing.T) {
	t.Parallel()
	cache := NewMemoryReadCache()
	ctx := t.Context()
	tenant := "tenant-a"
	email := "Agent@Example.TEST"

	cache.SetUnreadCount(ctx, tenant, email, 3)
	if count, ok := cache.GetUnreadCount(ctx, tenant, "agent@example.test"); !ok || count != 3 {
		t.Fatalf("expected cached unread 3, got count=%d ok=%v", count, ok)
	}
	payload := map[string]any{"items": []any{}, "next_before_at": ""}
	cache.SetList(ctx, tenant, email, "unread", "", 50, payload)
	if _, ok := cache.GetList(ctx, tenant, email, "unread", "", 50); !ok {
		t.Fatal("expected cached list")
	}

	cache.InvalidateUser(ctx, tenant, email)
	if _, ok := cache.GetUnreadCount(ctx, tenant, email); ok {
		t.Fatal("unread must miss after InvalidateUser")
	}
	if _, ok := cache.GetList(ctx, tenant, email, "unread", "", 50); ok {
		t.Fatal("list must miss after InvalidateUser")
	}
}
