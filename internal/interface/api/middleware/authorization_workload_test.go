package middleware

import (
	"os"
	"strings"
	"testing"
)

func TestWorkloadRolesIncludeAgentAndCallCenter(t *testing.T) {
	// anti-regresion: BUG-0971 — Cobranza progress bar needs agent/call_center workload.
	source, err := os.ReadFile("authorization.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(source)
	for _, fragment := range []string{
		`"agent", "call_center", "gestor_senior"`,
		`"manager", "sub_gerente", "admin"`,
		"EnforceWorkloadRoles",
		"BUG-0971",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("authorization must preserve %q", fragment)
		}
	}
}
