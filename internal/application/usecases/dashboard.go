package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// DashboardUsecase handles dashboard-related business logic.
type DashboardUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewDashboardUsecase creates a new DashboardUsecase.
func NewDashboardUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *DashboardUsecase {
	return &DashboardUsecase{core: core, crypto: crypto}
}

// GetSummary gets the dashboard summary.
func (u *DashboardUsecase) GetSummary(ctx context.Context, tenantID, traceID, userEmail string) (map[string]any, error) {
	return u.core.ListActivities(ctx, traceID, tenantID, "", nil, "", "", "", "", userEmail, 1, 0)
}

// GetAlerts gets the dashboard alerts.
// Python BFF wraps escalations items into {"data":{"alerts":[...]}}. Match that shape.
func (u *DashboardUsecase) GetAlerts(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	result, err := u.core.ListActivities(ctx, traceID, tenantID, "", nil, "", "", "", "escalation", userEmail, 50, 0)
	if err != nil {
		return nil, err
	}
	items, _ := result["items"].([]any)
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"data": map[string]any{
			"alerts": items,
		},
	}, nil
}
