package usecases

import (
	"context"

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
func (u *PaymentPromisesUsecase) ListPaymentPromises(ctx context.Context, loanID, clientID, agentID, agentName string, limit, offset int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.ListActivities(ctx, traceID, tenantID, loanID, nil, clientID, agentID, agentName, "payment_promise", userEmail, limit, offset)
}
