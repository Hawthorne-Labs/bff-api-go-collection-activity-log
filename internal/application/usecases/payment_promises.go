package usecases

import (
	"context"
	"fmt"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// PaymentPromisesUsecase handles payment promise-related business logic.
type PaymentPromisesUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewPaymentPromisesUsecase creates a new PaymentPromisesUsecase.
func NewPaymentPromisesUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *PaymentPromisesUsecase {
	return &PaymentPromisesUsecase{core: core, crypto: crypto}
}

// ListPaymentPromises lists payment promises with filtering.
func (u *PaymentPromisesUsecase) ListPaymentPromises(ctx context.Context, loanID, clientID, agentID string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	params := map[string]string{
		"limit":  fmt.Sprintf("%d", limit),
		"offset": fmt.Sprintf("%d", offset),
	}
	if loanID != "" {
		params["loan_id"] = loanID
	}
	if clientID != "" {
		params["client_id"] = clientID
	}
	if agentID != "" {
		params["agent_id"] = agentID
	}
	return u.core.ListActivities(ctx, traceID, tenantID, loanID, clientID, agentID, "", "payment_promise", userEmail, limit, offset)
}
