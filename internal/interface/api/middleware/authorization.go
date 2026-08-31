package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
)

var (
	allRoles = []string{
		"agent", "call_center", "gestor_senior", "supervisor", "especial",
		"manager", "sub_gerente", "admin", "auditor",
	}
	// anti-regresion: BUG-1019 ver handoffs/regressions.md (no revertir sin leer)
	supervisorRoles = []string{"supervisor", "manager", "admin", "sub_gerente", "especial"}
	// anti-regresion: BUG-0971 — agent/call_center need workload for Cobranza progress bar.
	workloadRoles = []string{
		"agent", "call_center", "gestor_senior", "supervisor", "especial",
		"manager", "sub_gerente", "admin",
	}
)

func RequireScope(c *gin.Context, required string) (*CognitoContext, bool) {
	ctx := GetCognitoContext(c)
	if ctx == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return nil, false
	}
	for scope := range scopesSet(ctx.Scope) {
		if scope == required {
			return ctx, true
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": map[string]any{"code": domain.AccessDenied, "message": "No tiene permisos para esta operación."}})
	return nil, false
}

func EnforceRole(c *gin.Context, allowed ...string) (*CognitoContext, bool) {
	ctx := GetCognitoContext(c)
	if ctx == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return nil, false
	}
	if slices.Contains(allowed, ctx.Role) {
		return ctx, true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": map[string]any{"code": domain.AccessDenied, "message": "No tiene permisos para esta operación."}})
	return nil, false
}

func EnforceAllRoles(c *gin.Context) (*CognitoContext, bool) {
	return EnforceRole(c, allRoles...)
}

func EnforceSupervisorRoles(c *gin.Context) (*CognitoContext, bool) {
		return EnforceRole(c, supervisorRoles...)
	}

	func EnforceWorkloadRoles(c *gin.Context) (*CognitoContext, bool) {
		return EnforceRole(c, workloadRoles...)
	}

func ResolveAgentID(ctx *CognitoContext, requested string) string {
	if ctx == nil {
		return strings.TrimSpace(requested)
	}
	// anti-regresion: BUG-0945 ver handoffs/regressions/BUG-0945-bff-resolve-agent-id-call-center.md (no revertir sin leer)
	if ctx.Role == "agent" || ctx.Role == "call_center" || ctx.Role == "gestor_senior" {
		return ctx.Sub
	}
	if trimmed := strings.TrimSpace(requested); trimmed != "" {
		return trimmed
	}
	return ctx.Sub
}

func scopesSet(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, scope := range strings.Fields(raw) {
		out[scope] = struct{}{}
	}
	return out
}
