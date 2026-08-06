package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
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
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 500 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	loanID := c.Query("loan_id")
	clientID := c.Query("client_id")
	agentID := c.Query("agent_id")
	agentName := c.Query("agent_name")
	activityType := c.Query("activity_type")

	result, err := h.activities.ListActivities(c.Request.Context(), loanID, clientID, agentID, agentName, activityType, limit, offset, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ActivitiesListFailed, "message": "No se pudo cargar la lista de actividades."}})
		return
	}

	c.JSON(http.StatusOK, result)
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
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ActivityCreateFailed, "message": "No se pudo crear la actividad."}})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// CreateActivityBatch handles POST /api/v1/collections/activities/batch
func (h *ActivitiesHandler) CreateActivityBatch(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.ActivityBatchFailed, "message": "Payload inválido."}})
		return
	}

	requestID := c.GetHeader("X-Request-Id")
	correlationID := c.GetHeader("X-Correlation-Id")
	traceparent := c.GetHeader("traceparent")
	idempotencyKey := c.GetHeader("Idempotency-Key")

	result, err := h.activities.CreateActivityBatch(c.Request.Context(), payload, traceID.(string), tenantID.(string), requestID, correlationID, traceparent, idempotencyKey, ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ActivityBatchFailed, "message": "No se pudo crear las actividades en lote."}})
		return
	}

	c.JSON(http.StatusCreated, result)
}
