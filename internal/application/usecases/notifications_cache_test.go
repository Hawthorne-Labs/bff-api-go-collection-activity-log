package usecases

import (
	"context"
	"testing"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/notifications"
)

type countingNotificationCache struct {
	inner          *notifications.MemoryReadCache
	unreadHits     int
	listHits       int
	invalidations  int
	setUnreadCalls int
	setListCalls   int
}

func (c *countingNotificationCache) GetUnreadCount(ctx context.Context, tenantID, userEmail string) (int, bool) {
	count, ok := c.inner.GetUnreadCount(ctx, tenantID, userEmail)
	if ok {
		c.unreadHits++
	}
	return count, ok
}

func (c *countingNotificationCache) SetUnreadCount(ctx context.Context, tenantID, userEmail string, count int) {
	c.setUnreadCalls++
	c.inner.SetUnreadCount(ctx, tenantID, userEmail, count)
}

func (c *countingNotificationCache) GetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int) (map[string]any, bool) {
	payload, ok := c.inner.GetList(ctx, tenantID, userEmail, state, severity, limit)
	if ok {
		c.listHits++
	}
	return payload, ok
}

func (c *countingNotificationCache) SetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int, payload map[string]any) {
	c.setListCalls++
	c.inner.SetList(ctx, tenantID, userEmail, state, severity, limit, payload)
}

func (c *countingNotificationCache) InvalidateUser(ctx context.Context, tenantID, userEmail string) {
	c.invalidations++
	c.inner.InvalidateUser(ctx, tenantID, userEmail)
}

// anti-regresion: BUG-1071 — unread-count must hit short-TTL cache on repeated polls
func TestUnreadCountServesFromCacheAside(t *testing.T) {
	t.Parallel()
	cache := &countingNotificationCache{inner: notifications.NewMemoryReadCache()}
	uc := &NotificationsUsecase{cache: cache}
	ctx := context.Background()
	tenant := "tenant-a"
	email := "agent@example.test"

	cache.SetUnreadCount(ctx, tenant, email, 7)
	got, err := uc.UnreadCount(ctx, "trace", tenant, email)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if got["count"] != 7 {
		t.Fatalf("expected count 7 from cache, got %#v", got)
	}
	if cache.unreadHits != 1 {
		t.Fatalf("expected one unread cache hit, got %d", cache.unreadHits)
	}
}

// anti-regresion: BUG-1071 — first-page list poll must hit cache without Core
func TestListNotificationsServesCacheableFirstPageFromCache(t *testing.T) {
	t.Parallel()
	cache := &countingNotificationCache{inner: notifications.NewMemoryReadCache()}
	uc := &NotificationsUsecase{cache: cache}
	ctx := context.Background()
	tenant := "tenant-a"
	email := "agent@example.test"
	payload := map[string]any{"items": []any{map[string]any{"id": "n1"}}, "next_before_at": ""}
	cache.SetList(ctx, tenant, email, "unread", "", 50, payload)

	got, err := uc.ListNotifications(ctx, "unread", "", "", "", 50, "", "", "trace", tenant, email)
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if cache.listHits != 1 {
		t.Fatalf("expected list cache hit, got %d", cache.listHits)
	}
	items, _ := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected cached items, got %#v", got)
	}
}

func TestUnreadCountFromCoreCoercesTypes(t *testing.T) {
	t.Parallel()
	if unreadCountFromCore(map[string]any{"count": float64(4)}) != 4 {
		t.Fatal("float64")
	}
	if unreadCountFromCore(map[string]any{"count": int64(2)}) != 2 {
		t.Fatal("int64")
	}
	if unreadCountFromCore(nil) != 0 {
		t.Fatal("nil")
	}
}
