package usecases

import (
	"context"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// NotificationsUsecase handles notification-related business logic.
type NotificationsUsecase struct {
	core   *coreclient.CoreClient
	crypto *cryptobffclient.CryptoBFFClient
}

// NewNotificationsUsecase creates a new NotificationsUsecase.
func NewNotificationsUsecase(core *coreclient.CoreClient, crypto *cryptobffclient.CryptoBFFClient) *NotificationsUsecase {
	return &NotificationsUsecase{core: core, crypto: crypto}
}

// ListNotifications lists notifications with filtering and pagination.
func (u *NotificationsUsecase) ListNotifications(ctx context.Context, state, severity, fromDate, toDate string, limit int, beforeAt, beforeID string, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.ListNotifications(ctx, traceID, tenantID, userEmail, state, severity, fromDate, toDate, limit, beforeAt, beforeID)
}

// NotificationEventsAfter gets notification events after a given event ID.
func (u *NotificationsUsecase) NotificationEventsAfter(ctx context.Context, afterEventID string, limit int, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.NotificationEventsAfter(ctx, traceID, tenantID, userEmail, afterEventID, limit)
}

// UnreadCount gets the unread notification count.
func (u *NotificationsUsecase) UnreadCount(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	return u.core.NotificationUnreadCount(ctx, traceID, tenantID, userEmail)
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
	return u.core.MarkAllNotificationsRead(ctx, traceID, tenantID, userEmail)
}

// MarkRead marks a single notification as read.
func (u *NotificationsUsecase) MarkRead(ctx context.Context, notificationID, traceID, tenantID, userEmail string) error {
	return u.core.MarkNotificationRead(ctx, notificationID, traceID, tenantID, userEmail)
}
