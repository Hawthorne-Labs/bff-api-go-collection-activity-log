package usecases

import (
	"context"
	"fmt"
)

const createLoanActivityPath = "/api/v1/collections/loans/{loanId}/activities"

func (u *ActivitiesUsecase) EncryptActivityPII(
	ctx context.Context,
	payload map[string]any,
	tenantID, requestID, correlationID, traceparent string,
) (map[string]any, error) {
	comment, ok := stringField(payload, "comment")
	if !ok || comment == "" {
		return payload, nil
	}
	encrypted, err := u.crypto.EncryptFields(
		ctx,
		"POST",
		createLoanActivityPath,
		map[string]string{"comment": comment},
		tenantID,
		requestID,
		correlationID,
		traceparent,
	)
	if err != nil {
		return nil, fmt.Errorf("encrypt activity comment: %w", err)
	}
	out := copyAnyMap(payload)
	for field, value := range encrypted {
		out[field] = value
	}
	return out, nil
}
