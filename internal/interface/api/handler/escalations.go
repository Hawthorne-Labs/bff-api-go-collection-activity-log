package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// EscalationsHandler handles all escalation-related HTTP endpoints.
type EscalationsHandler struct {
	escalations *usecases.EscalationsUsecase
}

// NewEscalationsHandler creates a new EscalationsHandler.
func NewEscalationsHandler(escalations *usecases.EscalationsUsecase) *EscalationsHandler {
	return &EscalationsHandler{escalations: escalations}
}

// ListEscalations handles GET /api/v1/collections/escalations
func (h *EscalationsHandler) ListEscalations(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	if _, ok := middleware.EnforceAllRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	loanID := c.Query("loan_id")
	clientID := c.Query("client_id")
	agentID := c.Query("agent_id")
	agentName := c.Query("agent_name")
	status := c.Query("status")

	result, err := h.escalations.ListEscalations(c.Request.Context(), loanID, clientID, agentID, agentName, status, limit, offset, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.EscalationListFailed, "No se pudo cargar la lista de escalamientos.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateEscalation handles POST /api/v1/collections/escalations
func (h *EscalationsHandler) CreateEscalation(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:write"); !ok {
		return
	}
	if _, ok := middleware.EnforceAllRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.EscalationCreateFailed, "message": "Payload inválido."}})
		return
	}

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")
	idempotencyKey := c.GetHeader("Idempotency-Key")

	result, err := h.escalations.CreateEscalation(c.Request.Context(), payload, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, idempotencyKey, ctx.Email)
	if err != nil {
		writeErr(c, err, domain.EscalationCreateFailed, "No se pudo crear el escalamiento.")
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CreateLoanEscalation handles POST /api/v1/collections/loans/:loanId/escalations
func (h *EscalationsHandler) CreateLoanEscalation(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	loanID := c.Param("loanId")
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.EscalationCreateFailed, "message": "Payload inválido."}})
		return
	}

	// Add loan_id to payload if not present
	if _, ok := payload["loan_id"]; !ok {
		payload["loan_id"] = loanID
	}

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")
	idempotencyKey := c.GetHeader("Idempotency-Key")

	result, err := h.escalations.CreateEscalation(c.Request.Context(), payload, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, idempotencyKey, ctx.Email)
	if err != nil {
		writeErr(c, err, domain.EscalationCreateFailed, "No se pudo crear el escalamiento del préstamo.")
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateEscalationStatus handles PATCH /api/v1/collections/escalations/:id/status
func (h *EscalationsHandler) UpdateEscalationStatus(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:write"); !ok {
		return
	}
	ctx, ok := middleware.EnforceSupervisorRoles(c)
	if !ok {
		return
	}

	escalationID := c.Param("id")
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.EscalationUpdateFailed, "message": "Payload inválido."}})
		return
	}

	result, err := h.escalations.UpdateEscalationStatus(c.Request.Context(), escalationID, payload, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.EscalationUpdateFailed, "No se pudo actualizar el estado del escalamiento.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// DecideEscalation handles POST /api/v1/collections/escalations/:id/decisions
func (h *EscalationsHandler) DecideEscalation(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:write"); !ok {
		return
	}
	ctx, ok := middleware.EnforceSupervisorRoles(c)
	if !ok {
		return
	}

	escalationID := c.Param("id")
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.EscalationDecideFailed, "message": "Payload inválido."}})
		return
	}

	result, err := h.escalations.DecideEscalation(c.Request.Context(), escalationID, payload, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.EscalationDecideFailed, "No se pudo registrar la decisión del escalamiento.")
		return
	}

	c.JSON(http.StatusOK, result)
}
