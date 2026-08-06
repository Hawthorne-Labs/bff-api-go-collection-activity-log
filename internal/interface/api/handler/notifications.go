package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// NotificationsHandler handles all notification-related HTTP endpoints.
type NotificationsHandler struct {
	notifications *usecases.NotificationsUsecase
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(notifications *usecases.NotificationsUsecase) *NotificationsHandler {
	return &NotificationsHandler{notifications: notifications}
}

// ListNotifications handles GET /api/v1/notifications
func (h *NotificationsHandler) ListNotifications(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	state := c.Query("state")
	severity := c.Query("severity")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	beforeID := c.Query("before_id")

	if limit < 1 || limit > 500 {
		limit = 20
	}

	result, err := h.notifications.ListNotifications(c.Request.Context(), state, severity, fromDate, toDate, limit, beforeID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "No se pudo cargar la lista de notificaciones."}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// NotificationEventsAfter handles GET /api/v1/notifications/stream
func (h *NotificationsHandler) NotificationEventsAfter(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	afterEventID := c.Query("after_event_id")

	if limit < 1 || limit > 500 {
		limit = 20
	}

	result, err := h.notifications.NotificationEventsAfter(c.Request.Context(), afterEventID, limit, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "No se pudo cargar los eventos de notificaciones."}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UnreadCount handles GET /api/v1/notifications/unread-count
func (h *NotificationsHandler) UnreadCount(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	result, err := h.notifications.UnreadCount(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "No se pudo obtener el conteo de no leídas."}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RegisterDevice handles PUT /api/v1/notifications/devices/current
func (h *NotificationsHandler) RegisterDevice(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.NotificationRegisterFailed, "message": "Payload inválido."}})
		return
	}

	result, err := h.notifications.RegisterDevice(c.Request.Context(), payload, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationRegisterFailed, "message": "No se pudo registrar el dispositivo."}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RevokeDevice handles DELETE /api/v1/notifications/devices/current
func (h *NotificationsHandler) RevokeDevice(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.NotificationRevokeFailed, "message": "Payload inválido."}})
		return
	}

	installationID, _ := payload["installation_id"].(string)
	if installationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": domain.NotificationRevokeFailed, "message": "Se requiere installation_id."}})
		return
	}

	err := h.notifications.RevokeDevice(c.Request.Context(), installationID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationRevokeFailed, "message": "No se pudo revocar el dispositivo."}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"revoked": true}})
}

// GetDetail handles GET /api/v1/notifications/:id
func (h *NotificationsHandler) GetDetail(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	notificationID := c.Param("id")
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	result, err := h.notifications.GetDetail(c.Request.Context(), notificationID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationDetailFailed, "message": "No se pudo obtener el detalle de la notificación."}})
		return
	}

	c.JSON(http.StatusOK, result)
}

// MarkAllRead handles POST /api/v1/notifications/read-all
func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	err := h.notifications.MarkAllRead(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationReadFailed, "message": "No se pudieron marcar las notificaciones como leídas."}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"marked_read": true}})
}

// MarkRead handles POST /api/v1/notifications/:id/read
func (h *NotificationsHandler) MarkRead(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	notificationID := c.Param("id")
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	err := h.notifications.MarkRead(c.Request.Context(), notificationID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationReadFailed, "message": "No se pudo marcar la notificación como leída."}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"marked_read": true}})
}
