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
	return u.core.ListActivities(ctx, traceID, tenantID, "", "", "", "", "", userEmail, 1, 0)
}

// GetAlerts gets the dashboard alerts.
func (u *DashboardUsecase) GetAlerts(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	// Alerts are derived from escalations and overdue payment promises.
	return u.core.ListEscalations(ctx, traceID, tenantID, "", "", "", "", "", userEmail, 50, 0)
}
