package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// ContactsUsecase handles contact submission business logic.
type ContactsUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewContactsUsecase creates a new ContactsUsecase.
func NewContactsUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *ContactsUsecase {
	return &ContactsUsecase{core: core, crypto: crypto}
}

// SubmitContact submits a contact form to the core.
func (u *ContactsUsecase) SubmitContact(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent string) (map[string]any, error) {
	return u.core.SubmitContact(ctx, payload, traceID, tenantID, requestID, correlationID, traceparent)
}
