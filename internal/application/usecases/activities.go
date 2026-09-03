package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/activities"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

type activityReadCache interface {
	GetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) (map[string]any, bool)
	SetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, payload map[string]any)
	InvalidateScopes(ctx context.Context, tenantID, loanID, clientID string)
}

// ActivitiesUsecase handles activity-related business logic.
type ActivitiesUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
	cache  activityReadCache
}

// NewActivitiesUsecase creates a new ActivitiesUsecase.
func NewActivitiesUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient, cache ...activityReadCache) *ActivitiesUsecase {
	var readCache activityReadCache
	if len(cache) > 0 {
		readCache = cache[0]
	}
	return &ActivitiesUsecase{core: core, crypto: crypto, cache: readCache}
}

// ListActivities lists activities with filtering.
// anti-regresion: BUG-1072 ver handoffs/regressions.md (no revertir sin leer)
func (u *ActivitiesUsecase) ListActivities(ctx context.Context, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	if u.cache != nil && activities.CacheableList(loanID, loanIDs, clientID, offset) {
		if cached, ok := u.cache.GetList(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset); ok {
			return cached, nil
		}
	}
	result, err := u.core.ListActivities(ctx, traceID, tenantID, loanID, loanIDs, clientID, agentID, agentName, activityType, userEmail, limit, offset)
	if err != nil {
		return nil, err
	}
	if u.cache != nil && activities.CacheableList(loanID, loanIDs, clientID, offset) {
		u.cache.SetList(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset, result)
	}
	return result, nil
}

// CreateActivity creates a single activity.
func (u *ActivitiesUsecase) CreateActivity(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	result, err := u.core.CreateActivity(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
	if err != nil {
		return nil, err
	}
	u.invalidateFromPayload(ctx, tenantID, payload)
	return result, nil
}

// CreateActivityBatch creates activities in batch.
func (u *ActivitiesUsecase) CreateActivityBatch(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	result, err := u.core.CreateActivityBatch(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
	if err != nil {
		return nil, err
	}
	u.invalidateFromPayload(ctx, tenantID, payload)
	if activity, ok := payload["activity"].(map[string]any); ok {
		u.invalidateFromPayload(ctx, tenantID, activity)
	}
	if rawIDs, ok := payload["loan_ids"].([]any); ok {
		for _, raw := range rawIDs {
			if id, ok := raw.(string); ok {
				u.invalidateScopes(ctx, tenantID, id, "")
			}
		}
	}
	return result, nil
}

func (u *ActivitiesUsecase) invalidateFromPayload(ctx context.Context, tenantID string, payload map[string]any) {
	loanID, clientID := activities.ScopeIDsFromPayload(payload)
	u.invalidateScopes(ctx, tenantID, loanID, clientID)
}

func (u *ActivitiesUsecase) invalidateScopes(ctx context.Context, tenantID, loanID, clientID string) {
	if u.cache == nil {
		return
	}
	u.cache.InvalidateScopes(ctx, tenantID, loanID, clientID)
}
