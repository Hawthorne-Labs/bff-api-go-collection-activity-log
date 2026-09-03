package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/notifications"
)

// notificationReadCache stores short-lived unread/list responses to cut Core/RDS load.
type notificationReadCache interface {
	GetUnreadCount(ctx context.Context, tenantID, userEmail string) (int, bool)
	SetUnreadCount(ctx context.Context, tenantID, userEmail string, count int)
	GetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int) (map[string]any, bool)
	SetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int, payload map[string]any)
	InvalidateUser(ctx context.Context, tenantID, userEmail string)
}

// NotificationsUsecase handles notification-related business logic.
type NotificationsUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
	cache  notificationReadCache
}

// NewNotificationsUsecase creates a new NotificationsUsecase.
func NewNotificationsUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient, cache ...notificationReadCache) *NotificationsUsecase {
	var readCache notificationReadCache
	if len(cache) > 0 {
		readCache = cache[0]
	}
	return &NotificationsUsecase{core: core, crypto: crypto, cache: readCache}
}

// ListNotifications lists notifications with filtering and pagination.
// anti-regresion: BUG-1071 ver handoffs/regressions.md (no revertir sin leer)
func (u *NotificationsUsecase) ListNotifications(ctx context.Context, state, severity, fromDate, toDate string, limit int, beforeAt, beforeID string, traceID, tenantID, userEmail string) (map[string]any, error) {
	if u.cache != nil && notifications.CacheableListPage(state, severity, fromDate, toDate, beforeAt, beforeID) {
		if cached, ok := u.cache.GetList(ctx, tenantID, userEmail, state, severity, limit); ok {
			return cached, nil
		}
	}
	result, err := u.core.ListNotifications(ctx, traceID, tenantID, userEmail, state, severity, fromDate, toDate, limit, beforeAt, beforeID)
	if err != nil {
		return nil, err
	}
	if u.cache != nil && notifications.CacheableListPage(state, severity, fromDate, toDate, beforeAt, beforeID) {
		u.cache.SetList(ctx, tenantID, userEmail, state, severity, limit, result)
	}
	return result, nil
}

// NotificationEventsAfter gets notification events after a given event ID.
func (u *NotificationsUsecase) NotificationEventsAfter(ctx context.Context, afterEventID string, limit int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.NotificationEventsAfter(ctx, traceID, tenantID, userEmail, afterEventID, limit)
}

// UnreadCount gets the unread notification count.
// anti-regresion: BUG-1071 ver handoffs/regressions.md (no revertir sin leer)
func (u *NotificationsUsecase) UnreadCount(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	if u.cache != nil {
		if count, ok := u.cache.GetUnreadCount(ctx, tenantID, userEmail); ok {
			return map[string]any{"count": count}, nil
		}
	}
	result, err := u.core.NotificationUnreadCount(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	if u.cache != nil {
		u.cache.SetUnreadCount(ctx, tenantID, userEmail, unreadCountFromCore(result))
	}
	return result, nil
}

// RegisterDevice registers a notification device.
func (u *NotificationsUsecase) RegisterDevice(ctx context.Context, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.RegisterNotificationDevice(ctx, payload, traceID, tenantID, userEmail)
}

// RevokeDevice revokes a notification device.
func (u *NotificationsUsecase) RevokeDevice(ctx context.Context, installationID, traceID, tenantID, userEmail string) error {
	return u.core.RevokeNotificationDevice(ctx, installationID, traceID, tenantID, userEmail)
}

// GetDetail gets a single notification detail.
func (u *NotificationsUsecase) GetDetail(ctx context.Context, notificationID, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.NotificationDetail(ctx, notificationID, traceID, tenantID, userEmail)
}

// MarkAllRead marks all notifications as read.
func (u *NotificationsUsecase) MarkAllRead(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	result, err := u.core.MarkAllNotificationsRead(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	if u.cache != nil {
		u.cache.InvalidateUser(ctx, tenantID, userEmail)
	}
	return result, nil
}

// MarkRead marks a single notification as read.
func (u *NotificationsUsecase) MarkRead(ctx context.Context, notificationID, traceID, tenantID, userEmail string) error {
	if err := u.core.MarkNotificationRead(ctx, notificationID, traceID, tenantID, userEmail); err != nil {
		return err
	}
	if u.cache != nil {
		u.cache.InvalidateUser(ctx, tenantID, userEmail)
	}
	return nil
}

func unreadCountFromCore(result map[string]any) int {
	if result == nil {
		return 0
	}
	switch v := result["count"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}
