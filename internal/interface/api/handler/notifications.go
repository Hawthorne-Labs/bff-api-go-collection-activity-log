package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/notifications"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// NotificationsHandler handles all notification-related HTTP endpoints.
type NotificationsHandler struct {
	notifications   *usecases.NotificationsUsecase
	cursors         *notificationCursorCodec
	redis           *goredis.Client
	streamMaxLifetime time.Duration
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(
	notifications *usecases.NotificationsUsecase,
	cursorSecret string,
	redisClient *goredis.Client,
	streamMaxSeconds int,
) *NotificationsHandler {
	if streamMaxSeconds < 15 {
		streamMaxSeconds = 300
	}
	return &NotificationsHandler{
		notifications: notifications,
		cursors:       newNotificationCursorCodec(cursorSecret),
		redis:         redisClient,
		streamMaxLifetime: time.Duration(streamMaxSeconds) * time.Second,
	}
}

// ListNotifications handles GET /api/v1/notifications
func (h *NotificationsHandler) ListNotifications(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "notifications:read"); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	state := c.DefaultQuery("state", "all")
	severity := c.Query("severity")
	fromDate := c.Query("from")
	if fromDate == "" {
		fromDate = c.Query("from_date")
	}
	toDate := c.Query("to")
	if toDate == "" {
		toDate = c.Query("to_date")
	}
	beforeAt := c.Query("before_at")
	beforeID := c.Query("before_id")
	if cursor := c.Query("cursor"); cursor != "" {
		decodedAt, decodedID, err := h.cursors.decode(cursor, ctx.Email, state, severity, fromDate, toDate)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "cursor de notificaciones inválido"}})
			return
		}
		beforeAt, beforeID = decodedAt, decodedID
	}

	if limit < 1 || limit > 100 {
		limit = 50
	}

	result, err := h.notifications.ListNotifications(c.Request.Context(), state, severity, fromDate, toDate, limit, beforeAt, beforeID, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "No se pudo cargar la lista de notificaciones."}})
		return
	}

	mapped := make([]map[string]any, 0)
	for _, item := range notificationItems(result) {
		mapped = append(mapped, mapNotificationItem(item))
	}
	var nextCursor any
	nextBeforeAt := notificationString(result, "next_before_at", "NextBeforeAt")
	nextBeforeID := notificationString(result, "next_before_id", "NextBeforeID")
	if nextBeforeAt != "" && nextBeforeID != "" {
		nextCursor = h.cursors.encode(nextBeforeAt, nextBeforeID, ctx.Email, state, severity, fromDate, toDate)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": mapped, "next_cursor": nextCursor}, "meta": gin.H{}})
}

// NotificationEventsAfter handles GET /api/v1/notifications/stream
func (h *NotificationsHandler) NotificationEventsAfter(c *gin.Context) {
	if _, ok := middleware.RequireScope(c, "notifications:read"); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)
	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	afterEventID := c.GetHeader("Last-Event-ID")
	if afterEventID == "" {
		afterEventID = c.Query("after")
	}
	if afterEventID == "" {
		afterEventID = c.Query("after_event_id")
	}
	if afterEventID == "" {
		afterEventID = "0"
	}
	lastID, err := strconv.Atoi(afterEventID)
	if err != nil || lastID < 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "Last-Event-ID inválido"}})
		return
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}

	result, err := h.notifications.NotificationEventsAfter(c.Request.Context(), strconv.Itoa(lastID), limit, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationsListFailed, "message": "No se pudo cargar los eventos de notificaciones."}})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	recovered := notificationItems(result)
	recoverAfter := func(after int) ([]map[string]any, error) {
		response, recErr := h.notifications.NotificationEventsAfter(
			c.Request.Context(),
			strconv.Itoa(after),
			limit,
			traceID.(string),
			tenantID.(string),
			ctx.Email,
		)
		if recErr != nil {
			return nil, recErr
		}
		return notificationItems(response), nil
	}
	stream := notifications.NewSSE(h.redis, tenantID.(string), ctx.Email, recovered, lastID, recoverAfter, h.streamMaxLifetime)
	flusher, _ := c.Writer.(http.Flusher)
	flush := func() {
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := stream.Serve(c.Request.Context(), c.Writer, flush); err != nil && c.Request.Context().Err() == nil {
		_, _ = c.Writer.Write([]byte("event: retry\ndata: {\"reason\":\"stream_unavailable\"}\n\n"))
		flush()
	}
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

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": notificationCount(result)}, "meta": gin.H{}})
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

	_, err := h.notifications.RegisterDevice(c.Request.Context(), payload, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationRegisterFailed, "message": "No se pudo registrar el dispositivo."}})
		return
	}

	c.Status(http.StatusNoContent)
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

	installationID := c.Query("installation_id")
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

	c.Status(http.StatusNoContent)
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

	detail := result
	if nested := asObjectMap(result["data"]); nested != nil {
		detail = nested
	}
	c.JSON(http.StatusOK, gin.H{"data": mapNotificationDetail(detail), "meta": gin.H{}})
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

	result, err := h.notifications.MarkAllRead(c.Request.Context(), traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.NotificationReadFailed, "message": "No se pudieron marcar las notificaciones como leídas."}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": notificationCount(result)}, "meta": gin.H{}})
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

	c.Status(http.StatusNoContent)
}
