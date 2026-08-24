package usecases

import "testing"

func TestPrepareCreatePayloadKeepsFLEDecryptedReasonForCore(t *testing.T) {
	usecase := &EscalationsUsecase{}
	payload, err := usecase.prepareCreatePayload(
		t.Context(),
		map[string]any{
			"loan_id":                    "loan-1",
			"notes":                      "Motivo de escalamiento",
			"last_effective_activity_id": "activity-1",
		},
		"trace-1", "tenant-1", "request-1", "correlation-1", "", "agent@example.test",
	)
	if err != nil {
		t.Fatalf("prepare escalation payload: %v", err)
	}
	if got, want := payload["reason"], "Motivo de escalamiento"; got != want {
		t.Fatalf("reason = %v, want %q", got, want)
	}
}

func TestIsManagementActivitySkipsEscalationRows(t *testing.T) {
	if isManagementActivity(map[string]any{"activity_type": "escalation", "agent_name": "Agente"}) {
		t.Fatal("escalation rows must not count as management activity")
	}
}

func TestIsEscalationManagementActivityAllowsUnansweredHumanManagement(t *testing.T) {
	if !isEscalationManagementActivity(map[string]any{"activity_type": "call", "agent_name": "Agente", "answered": false}) {
		t.Fatal("an unanswered human management must permit escalation")
	}
	if isEscalationManagementActivity(map[string]any{"activity_type": "payment", "agent_name": "Agente"}) {
		t.Fatal("payment rows must not count as human management")
	}
	if isEscalationManagementActivity(map[string]any{"activity_type": "call", "agent_name": "Sistema"}) {
		t.Fatal("system rows must not count as human management")
	}
}
