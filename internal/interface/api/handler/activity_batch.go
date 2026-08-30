package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
)

const (
	batchMinLoans = 2
	batchMaxLoans = 500
)

// normalizeActivityBatchPayload validates and normalizes batch create payload like the Python BFF.
// Returns normalized payload (sorted unique loan_ids + activity without nested loan_id) or a BusinessError.
func normalizeActivityBatchPayload(payload map[string]any) (map[string]any, *domain.BusinessError) {
	if payload == nil {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "Validacion fallida.")
	}
	rawIDs, ok := payload["loan_ids"]
	if !ok {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "loan_ids es requerido.")
	}
	loanIDs, err := coerceStringSlice(rawIDs)
	if err != nil {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "loan_ids debe ser una lista de textos.")
	}
	normalized := make([]string, 0, len(loanIDs))
	seen := map[string]struct{}{}
	for _, id := range loanIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "loan_ids must be unique and non-blank")
		}
		if _, exists := seen[trimmed]; exists {
			return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "loan_ids must be unique and non-blank")
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) < batchMinLoans || len(normalized) > batchMaxLoans {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, fmt.Sprintf("loan_ids must contain between %d and %d items", batchMinLoans, batchMaxLoans))
	}
	sort.Strings(normalized)

	activityRaw, ok := payload["activity"]
	if !ok || activityRaw == nil {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "activity es requerido.")
	}
	activity, ok := activityRaw.(map[string]any)
	if !ok {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "activity debe ser un objeto.")
	}
	if _, hasLoan := activity["loan_id"]; hasLoan && activity["loan_id"] != nil {
		return nil, domain.NewHTTPBusinessError(http.StatusUnprocessableEntity, domain.ValidationError, "activity.loan_id is not allowed for batch management")
	}

	missing := []string{}
	if strings.TrimSpace(stringField(activity, "result")) == "" {
		missing = append(missing, "result")
	}
	if strings.TrimSpace(stringField(activity, "comment")) == "" {
		missing = append(missing, "comment")
	}
	if len(missing) > 0 {
		return nil, domain.NewHTTPBusinessError(
			http.StatusUnprocessableEntity,
			domain.ValidationError,
			fmt.Sprintf("required field(s) missing: %s", strings.Join(missing, ", ")),
		)
	}

	activityOut := cloneMap(activity)
	delete(activityOut, "loan_id")
	return map[string]any{
		"loan_ids": normalized,
		"activity": activityOut,
	}, nil
}

func coerceStringSlice(raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("non-string")
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("not a list")
	}
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
