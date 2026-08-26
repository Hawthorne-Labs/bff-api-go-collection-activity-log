package middleware

import (
	"os"
	"strings"
	"testing"
)

func TestResolveAgentIDTreatsCallCenterLikeAgent(t *testing.T) {
	// anti-regresion: BUG-0945 ver handoffs/regressions/BUG-0945-bff-resolve-agent-id-call-center.md (no revertir sin leer)
	source, err := os.ReadFile("authorization.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(source)
	if !strings.Contains(body, `ctx.Role == "agent" || ctx.Role == "call_center"`) {
		t.Fatal("ResolveAgentID must force own Sub for agent and call_center")
	}
	if !strings.Contains(body, "BUG-0945") {
		t.Fatal("ResolveAgentID must keep the BUG-0945 anti-regression marker")
	}
}

func TestResolveAgentIDForcesOwnSubForCallCenter(t *testing.T) {
	got := ResolveAgentID(&CognitoContext{Role: "call_center", Sub: "self-agent"}, "other-agent")
	if got != "self-agent" {
		t.Fatalf("call_center must not honor requested agent_id, got %q", got)
	}
	got = ResolveAgentID(&CognitoContext{Role: "agent", Sub: "self-agent"}, "other-agent")
	if got != "self-agent" {
		t.Fatalf("agent must not honor requested agent_id, got %q", got)
	}
	got = ResolveAgentID(&CognitoContext{Role: "supervisor", Sub: "sup"}, "other-agent")
	if got != "other-agent" {
		t.Fatalf("supervisor may request agent_id, got %q", got)
	}
}
