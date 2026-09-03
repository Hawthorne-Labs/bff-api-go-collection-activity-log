package usecases

import (
	"context"
	"testing"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/activities"
)

type countingActivityCache struct {
	inner *activities.MemoryReadCache
	hits  int
}

func (c *countingActivityCache) GetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) (map[string]any, bool) {
	payload, ok := c.inner.GetList(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset)
	if ok {
		c.hits++
	}
	return payload, ok
}

func (c *countingActivityCache) SetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, payload map[string]any) {
	c.inner.SetList(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset, payload)
}

func (c *countingActivityCache) InvalidateScopes(ctx context.Context, tenantID, loanID, clientID string) {
	c.inner.InvalidateScopes(ctx, tenantID, loanID, clientID)
}

// anti-regresion: BUG-1072 — scoped first-page activities served from cache
func TestListActivitiesServesScopedFirstPageFromCache(t *testing.T) {
	t.Parallel()
	cache := &countingActivityCache{inner: activities.NewMemoryReadCache()}
	uc := &ActivitiesUsecase{cache: cache}
	ctx := context.Background()
	payload := map[string]any{"items": []any{map[string]any{"id": "a1"}}, "total": 1}
	cache.SetList(ctx, "tenant", "agent@test", "loan-1", nil, "", "", "", "", 20, 0, payload)

	got, err := uc.ListActivities(ctx, "loan-1", nil, "", "", "", "", 20, 0, "trace", "tenant", "agent@test")
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if cache.hits != 1 {
		t.Fatalf("expected cache hit, got %d", cache.hits)
	}
	items, _ := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("cached payload missing, got %#v", got)
	}
}

func TestCreateActivityInvalidatesScopedCache(t *testing.T) {
	t.Parallel()
	cache := &countingActivityCache{inner: activities.NewMemoryReadCache()}
	uc := &ActivitiesUsecase{cache: cache}
	ctx := context.Background()
	cache.SetList(ctx, "tenant", "agent@test", "loan-1", nil, "", "", "", "", 20, 0, map[string]any{"items": []any{}})
	uc.invalidateFromPayload(ctx, "tenant", map[string]any{"loan_id": "loan-1"})
	if _, ok := cache.GetList(ctx, "tenant", "agent@test", "loan-1", nil, "", "", "", "", 20, 0); ok {
		t.Fatal("expected miss after create invalidate")
	}
}
