package usecases

import "testing"

func TestMapCreateEscalationPayloadUsesNotesAsReason(t *testing.T) {
	got := mapCreateEscalationPayload(map[string]any{
		"loan_id": "L1",
		"notes":   "Abonos\ncomentario",
	})
	if got["reason"] != "Abonos\ncomentario" {
		t.Fatalf("expected notes mapped to reason, got %#v", got)
	}
	if _, ok := got["notes"]; ok {
		t.Fatalf("notes must not be forwarded to core: %#v", got)
	}
}

func TestMapEscalationStatusPayloadNormalizesSpanishAliases(t *testing.T) {
	got := mapEscalationStatusPayload(map[string]any{"escalation_status": "pendiente"})
	if got["status"] != "PENDING" {
		t.Fatalf("expected PENDING, got %#v", got)
	}
	if _, ok := got["escalation_status"]; ok {
		t.Fatalf("escalation_status must not be forwarded: %#v", got)
	}
}
