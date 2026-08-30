package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// DashboardHandler handles dashboard-related HTTP endpoints.
type DashboardHandler struct {
	dashboard *usecases.DashboardUsecase
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashboard *usecases.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// GetSummary handles GET /api/v1/collections/dashboard/summary
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	result, err := h.dashboard.GetSummary(c.Request.Context(), tenantID.(string), traceID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.DashboardSummaryFailed, "No se pudo cargar el resumen del dashboard.")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAlerts handles GET /api/v1/collections/dashboard/alerts
func (h *DashboardHandler) GetAlerts(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	result, err := h.dashboard.GetAlerts(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		writeErr(c, err, domain.DashboardAlertsFailed, "No se pudieron cargar las alertas del dashboard.")
		return
	}

	c.JSON(http.StatusOK, result)
}
