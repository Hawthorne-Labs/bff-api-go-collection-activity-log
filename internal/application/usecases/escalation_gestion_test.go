package usecases

import "testing"

func TestIsManagementActivitySkipsEscalationRows(t *testing.T) {
	if isManagementActivity(map[string]any{"activity_type": "escalation", "agent_name": "Agente"}) {
		t.Fatal("escalation rows must not count as management activity")
	}
}

func TestIsEffectiveActivityRequiresAnsweredTrue(t *testing.T) {
	if !isEffectiveActivity(map[string]any{"answered": true}) {
		t.Fatal("answered=true must be effective")
	}
	if isEffectiveActivity(map[string]any{"answered": false}) {
		t.Fatal("answered=false must not be effective")
	}
}
