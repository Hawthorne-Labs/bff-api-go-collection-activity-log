package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// AgentPerformanceUsecase handles agent performance-related business logic.
type AgentPerformanceUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewAgentPerformanceUsecase creates a new AgentPerformanceUsecase.
func NewAgentPerformanceUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *AgentPerformanceUsecase {
	return &AgentPerformanceUsecase{core: core, crypto: crypto}
}

// GetKPIs gets agent KPIs for a given day.
func (u *AgentPerformanceUsecase) GetKPIs(ctx context.Context, agentID, day, traceID, tenantID string) (map[string]any, error) {
	return u.core.GetAgentKPIs(ctx, agentID, traceID, tenantID, day)
}

// GetGoals gets agent goals.
func (u *AgentPerformanceUsecase) GetGoals(ctx context.Context, agentID, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.GetAgentGoals(ctx, agentID, traceID, tenantID, userEmail)
}

// GetRanking gets the agent performance ranking.
func (u *AgentPerformanceUsecase) GetRanking(ctx context.Context, day string, limit int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.GetRanking(ctx, traceID, tenantID, day, limit, userEmail)
}

// GetWorkload gets agent workload.
func (u *AgentPerformanceUsecase) GetWorkload(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.GetWorkload(ctx, traceID, tenantID, userEmail)
}

// GetTeamReport gets the team performance report.
func (u *AgentPerformanceUsecase) GetTeamReport(ctx context.Context, traceID, tenantID, userEmail, tenant string) (map[string]any, error) {
	return u.core.GetTeamPerformanceReport(ctx, traceID, tenantID, userEmail, tenant)
}

// GetOperationsSummary gets the admin operations summary for a tenant/day.
func (u *AgentPerformanceUsecase) GetOperationsSummary(ctx context.Context, traceID, tenantID, userEmail, tenant, day string) (map[string]any, error) {
	return u.core.GetOperationsSummary(ctx, traceID, tenantID, userEmail, tenant, day)
}

// GetScopedGoals gets scoped goals for the current user/team.
func (u *AgentPerformanceUsecase) GetScopedGoals(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.GetScopedGoals(ctx, traceID, tenantID, userEmail)
}
