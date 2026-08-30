package handler

import (
	"net/http"
	"testing"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
)

func TestNormalizeActivityBatchPayloadSortsAndRejectsNestedLoanID(t *testing.T) {
	got, err := normalizeActivityBatchPayload(map[string]any{
		"loan_ids": []any{"loan-b", "loan-a"},
		"activity": map[string]any{
			"activity_type": "CALL",
			"result":        "CONTACTED",
			"comment":       "ok",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}
	ids := got["loan_ids"].([]string)
	if len(ids) != 2 || ids[0] != "loan-a" || ids[1] != "loan-b" {
		t.Fatalf("expected sorted loan_ids, got %#v", ids)
	}
}

func TestNormalizeActivityBatchPayloadRejectsNestedLoanID(t *testing.T) {
	_, err := normalizeActivityBatchPayload(map[string]any{
		"loan_ids": []any{"loan-a", "loan-b"},
		"activity": map[string]any{
			"loan_id": "loan-a",
			"result":  "CONTACTED",
			"comment": "ok",
		},
	})
	if err == nil || err.Code != domain.ValidationError || err.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error, got %#v", err)
	}
}

func TestNormalizeActivityBatchPayloadRequiresResultComment(t *testing.T) {
	_, err := normalizeActivityBatchPayload(map[string]any{
		"loan_ids": []any{"loan-a", "loan-b"},
		"activity": map[string]any{"activity_type": "CALL"},
	})
	if err == nil || err.Code != domain.ValidationError {
		t.Fatalf("expected validation error, got %#v", err)
	}
}

func TestNormalizeActivityBatchPayloadRejectsDuplicates(t *testing.T) {
	_, err := normalizeActivityBatchPayload(map[string]any{
		"loan_ids": []any{"loan-a", "loan-a"},
		"activity": map[string]any{"result": "CONTACTED", "comment": "ok"},
	})
	if err == nil {
		t.Fatal("expected duplicate rejection")
	}
}
