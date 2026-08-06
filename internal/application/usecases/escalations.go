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
	return u.core.CreateEscalation(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail)
}

// UpdateEscalationStatus updates the status of an escalation.
func (u *EscalationsUsecase) UpdateEscalationStatus(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.UpdateEscalationStatus(ctx, escalationID, payload, traceID, tenantID, userEmail)
}

// DecideEscalation records a decision on an escalation.
func (u *EscalationsUsecase) DecideEscalation(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.DecideEscalation(ctx, escalationID, payload, traceID, tenantID, userEmail)
}

// ListEscalations lists escalations with filtering.
func (u *EscalationsUsecase) ListEscalations(ctx context.Context, loanID, clientID, agentID, agentName, status string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.ListEscalations(ctx, traceID, tenantID, loanID, clientID, agentID, agentName, status, userEmail, limit, offset)
}
