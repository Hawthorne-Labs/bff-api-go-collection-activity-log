package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// AgentPerformanceHandler handles agent performance-related HTTP endpoints.
type AgentPerformanceHandler struct {
	agentPerformance *usecases.AgentPerformanceUsecase
}

// NewAgentPerformanceHandler creates a new AgentPerformanceHandler.
func NewAgentPerformanceHandler(agentPerformance *usecases.AgentPerformanceUsecase) *AgentPerformanceHandler {
	return &AgentPerformanceHandler{agentPerformance: agentPerformance}
}

// GetKPIs handles GET /api/v1/collections/agent-performance/kpis
func (h *AgentPerformanceHandler) GetKPIs(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	ctx, ok := middleware.EnforceAllRoles(c)
	if !ok {
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	agentID := middleware.ResolveAgentID(ctx, c.Query("agent_id"))
	day := c.Query("day")

	result, err := h.agentPerformance.GetKPIs(c.Request.Context(), agentID, day, traceID.(string), tenantID.(string))
	if err != nil {
		writeErr(c, err, domain.AgentKPIsFailed, "No se pudieron cargar los KPIs del agente.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetGoals handles GET /api/v1/collections/agent-performance/goals
func (h *AgentPerformanceHandler) GetGoals(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	ctx, ok := middleware.EnforceAllRoles(c)
	if !ok {
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	agentID := middleware.ResolveAgentID(ctx, c.Query("agent_id"))

	result, err := h.agentPerformance.GetGoals(c.Request.Context(), agentID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.AgentGoalsFailed, "No se pudieron cargar las metas del agente.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRanking handles GET /api/v1/collections/agent-performance/ranking
func (h *AgentPerformanceHandler) GetRanking(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	if _, ok := middleware.EnforceSupervisorRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	day := c.Query("day")

	if limit < 1 || limit > 100 {
		limit = 10
	}

	result, err := h.agentPerformance.GetRanking(c.Request.Context(), day, limit, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.AgentRankingFailed, "No se pudo cargar el ranking de agentes.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetWorkload handles GET /api/v1/collections/agent-performance/workload
func (h *AgentPerformanceHandler) GetWorkload(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	// anti-regresion: BUG-0971 — Cobranza progress bar for agent/call_center.
	ctx, ok := middleware.EnforceWorkloadRoles(c)
	if !ok {
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	result, err := h.agentPerformance.GetWorkload(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.AgentWorkloadFailed, "No se pudo cargar la carga de trabajo de agentes.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetReport handles GET /api/v1/collections/agent-performance/report
func (h *AgentPerformanceHandler) GetReport(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	ctx, ok := middleware.EnforceSupervisorRoles(c)
	if !ok {
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")
	tenant := strings.TrimSpace(c.Query("tenant"))
	if tenant == "" {
		tenant = strings.TrimSpace(c.GetHeader("X-Tenant-Id"))
	}

	result, err := h.agentPerformance.GetTeamReport(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email, tenant)
	if err != nil {
		writeErr(c, err, domain.AgentReportFailed, "No se pudo cargar el reporte de rendimiento del equipo.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOperationsSummary handles GET /api/v1/collections/agent-performance/operations-summary
func (h *AgentPerformanceHandler) GetOperationsSummary(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	if _, ok := middleware.EnforceAdminRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")
	tenant := strings.TrimSpace(c.Query("tenant"))
	day := strings.TrimSpace(c.Query("date"))

	result, err := h.agentPerformance.GetOperationsSummary(
		c.Request.Context(),
		traceID.(string),
		tenantID.(string),
		ctx.Email,
		tenant,
		day,
	)
	if err != nil {
		writeErr(c, err, domain.AgentOperationsSummaryFailed, "No se pudo cargar el resumen operativo por marca.")
		return
	}

	c.JSON(http.StatusOK, result)
}
