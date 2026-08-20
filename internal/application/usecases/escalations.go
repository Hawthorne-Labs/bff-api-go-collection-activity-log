package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// EscalationsUsecase handles escalation-related business logic.
type EscalationsUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewEscalationsUsecase creates a new EscalationsUsecase.
func NewEscalationsUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *EscalationsUsecase {
	return &EscalationsUsecase{core: core, crypto: crypto}
}

// CreateEscalation creates an escalation record.
func (u *EscalationsUsecase) CreateEscalation(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	prepared, err := u.prepareCreatePayload(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, userEmail)
	if err != nil {
		return nil, err
	}
	return u.core.CreateEscalation(ctx, prepared, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
}

// UpdateEscalationStatus updates the status of an escalation.
func (u *EscalationsUsecase) UpdateEscalationStatus(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.UpdateEscalationStatus(ctx, escalationID, mapEscalationStatusPayload(payload), traceID, tenantID, userEmail)
}

// DecideEscalation records a decision on an escalation.
func (u *EscalationsUsecase) DecideEscalation(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.DecideEscalation(ctx, escalationID, mapEscalationDecisionPayload(payload), traceID, tenantID, userEmail)
}

// ListEscalations lists escalations via the activities endpoint with activity_type=escalation filter.
// This matches the Python BFF behaviour: Core does not expose a standalone GET escalations route,
// so we delegate to the activities list which already applies visibility scoping.
func (u *EscalationsUsecase) ListEscalations(ctx context.Context, loanID, clientID, agentID, agentName, status string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.ListActivities(ctx, traceID, tenantID, loanID, nil, clientID, agentID, agentName, "escalation", userEmail, limit, offset)
}
