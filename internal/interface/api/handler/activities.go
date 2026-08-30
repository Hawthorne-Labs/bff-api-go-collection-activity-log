package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/idempotency"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// ActivitiesHandler handles all activity-related HTTP endpoints.
type ActivitiesHandler struct {
	activities *usecases.ActivitiesUsecase
}

// NewActivitiesHandler creates a new ActivitiesHandler.
func NewActivitiesHandler(activities *usecases.ActivitiesUsecase) *ActivitiesHandler {
	return &ActivitiesHandler{activities: activities}
}

// ListActivities handles GET /api/v1/collections/activities
func (h *ActivitiesHandler) ListActivities(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resolvedPage, resolvedPageSize, resolvedOffset := resolveActivityPage(page, pageSize, limit, offset, c.Query("limit") != "")

	loanID := c.Query("loan_id")
	if loanID == "" {
		loanID = c.Param("loanId")
	}
	clientID := c.Query("client_id")
	agentID := c.Query("agent_id")
	agentName := c.Query("agent_name")
	activityType := c.Query("activity_type")

	var loanIDs []string
	if raw := c.Query("loan_ids"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				loanIDs = append(loanIDs, trimmed)
			}
		}
	}

	result, err := h.activities.ListActivities(c.Request.Context(), loanID, loanIDs, clientID, agentID, agentName, activityType, resolvedPageSize, resolvedOffset, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.ActivitiesListFailed, "No se pudo cargar la lista de actividades.")
		return
	}

	c.JSON(http.StatusOK, paginateActivities(result, resolvedPage, resolvedPageSize))
}

// CreateActivity handles POST /api/v1/collections/activities
func (h *ActivitiesHandler) CreateActivity(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.ActivityCreateFailed, "message": "Payload inválido."}})
		return
	}

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")
	idempotencyKey := c.GetHeader("Idempotency-Key")

	result, err := h.activities.CreateActivity(c.Request.Context(), payload, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, idempotencyKey, ctx.Email)
	if err != nil {
		writeErr(c, err, domain.ActivityCreateFailed, "No se pudo crear la actividad.")
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CreateLoanActivity handles POST /api/v1/collections/loans/:loanId/activities
func (h *ActivitiesHandler) CreateLoanActivity(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:write"); !ok {
		return
	}
	if _, ok := middleware.EnforceAllRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	loanID := c.Param("loanId")
	if loanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.ActivityCreateFailed, "message": "loan_id es requerido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.ActivityCreateFailed, "message": "Payload invalido."}})
		return
	}
	payload["loan_id"] = loanID

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")
	idempotencyKey := c.GetHeader("Idempotency-Key")

	encryptedPayload, err := h.activities.EncryptActivityPII(
		c.Request.Context(),
		payload,
		tenantID.(string),
		requestID,
		correlationID,
		traceparent,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": map[string]any{"code": domain.ActivityCreateFailed, "message": "No se pudo cifrar la gestión."}})
		return
	}

	result, err := h.activities.CreateActivity(c.Request.Context(), encryptedPayload, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, idempotencyKey, ctx.Email)
	if err != nil {
		writeErr(c, err, domain.ActivityCreateFailed, "No se pudo crear la actividad.")
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CreateActivityBatch handles POST /api/v1/collections/activities/batch
func (h *ActivitiesHandler) CreateActivityBatch(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "collections:write"); !ok {
		return
	}
	ctx, ok := middleware.EnforceAllRoles(c)
	if !ok {
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		writeAPIError(c, http.StatusBadRequest, domain.ActivityBatchFailed, "Payload inválido.")
		return
	}

	normalized, bizErr := normalizeActivityBatchPayload(payload)
	if bizErr != nil {
		writeErr(c, bizErr, domain.ValidationError, bizErr.Message)
		return
	}

	clientKey, keyReason := idempotency.ValidateClientKey(c.GetHeader("Idempotency-Key"))
	if keyReason == "required" {
		writeAPIError(c, http.StatusBadRequest, domain.IdempotencyKeyRequired, "a valid Idempotency-Key header is required")
		return
	}
	if keyReason == "invalid" {
		writeAPIError(c, http.StatusBadRequest, domain.IdempotencyKeyInvalid, "a valid Idempotency-Key header is required")
		return
	}

	payloadHash, err := idempotency.CanonicalHash(normalized)
	if err != nil {
		writeAPIError(c, http.StatusBadRequest, domain.ActivityBatchFailed, "Payload inválido.")
		return
	}
	actor := ctx.Sub
	if actor == "" {
		actor = ctx.Email
	}
	if actor == "" {
		actor = "anonymous"
	}
	normalized["idempotency_key"] = clientKey
	normalized["payload_hash"] = payloadHash
	normalized["idempotency_scope"] = idempotency.ScopeKey("activity-log", actor, http.MethodPost, c.Request.URL.Path)

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")

	result, err := h.activities.CreateActivityBatch(c.Request.Context(), normalized, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, clientKey, ctx.Email)
	if err != nil {
		writeErr(c, err, domain.ActivityBatchFailed, "No se pudo crear las actividades en lote.")
		return
	}

	c.JSON(http.StatusCreated, result)
}

func resolveActivityPage(page, pageSize, limit, offset int, limitProvided bool) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	if !limitProvided {
		return page, pageSize, (page - 1) * pageSize
	}
	if limit < 1 || limit > 500 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return offset/limit + 1, limit, offset
}

func paginateActivities(response map[string]any, page, pageSize int) map[string]any {
	if response == nil {
		response = map[string]any{}
	}
	total := 0
	switch value := response["total"].(type) {
	case float64:
		total = int(value)
	case int:
		total = value
	case int64:
		total = int(value)
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	items := response["items"]
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"items":        items,
		"page":         page,
		"page_size":    pageSize,
		"total_items":  total,
		"total_pages":  totalPages,
		"has_next":     page < totalPages,
		"has_previous": page > 1,
		"total":        total,
	}
}
