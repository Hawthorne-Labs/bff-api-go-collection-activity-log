package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// ActivitiesUsecase handles activity-related business logic.
type ActivitiesUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewActivitiesUsecase creates a new ActivitiesUsecase.
func NewActivitiesUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *ActivitiesUsecase {
	return &ActivitiesUsecase{core: core, crypto: crypto}
}

// ListActivities lists activities with filtering.
func (u *ActivitiesUsecase) ListActivities(ctx context.Context, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.ListActivities(ctx, traceID, tenantID, loanID, loanIDs, clientID, agentID, agentName, activityType, userEmail, limit, offset)
}

// CreateActivity creates a single activity.
func (u *ActivitiesUsecase) CreateActivity(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	return u.core.CreateActivity(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
}

// CreateActivityBatch creates activities in batch.
func (u *ActivitiesUsecase) CreateActivityBatch(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	return u.core.CreateActivityBatch(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
}
