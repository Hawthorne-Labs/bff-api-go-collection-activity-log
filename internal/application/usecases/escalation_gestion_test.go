package usecases

import "testing"

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
