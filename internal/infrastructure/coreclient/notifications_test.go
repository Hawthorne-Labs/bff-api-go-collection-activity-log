package coreclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
)

func TestListNotificationsUsesCoreQueryNames(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewCoreClient(&config.Config{
		CoreBaseURL:             server.URL,
		RequestTimeoutSeconds:   5,
		InternalJWTSecret:       "dev-internal-jwt-secret-32-bytes-min",
		InternalJWTIssuer:       "python-templates-finch",
		InternalJWTCoreAudience: "core-api",
	})
	_, err := client.ListNotifications(context.Background(), "t", "COGASA", "emilio@hawthornelabs.io", "unread", "", "2026-01-01T00:00:00Z", "", 50, "", "orphan-id")
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if !strings.Contains(gotQuery, "state=unread") || !strings.Contains(gotQuery, "from=2026-01-01") {
		t.Fatalf("expected from/state params, got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "from_date=") || strings.Contains(gotQuery, "before_id=") {
		t.Fatalf("must not send from_date or unpaired before_id, got %q", gotQuery)
	}
}

func TestNotificationEventsAfterAlwaysSendsAfter(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewCoreClient(&config.Config{
		CoreBaseURL:             server.URL,
		RequestTimeoutSeconds:   5,
		InternalJWTSecret:       "dev-internal-jwt-secret-32-bytes-min",
		InternalJWTIssuer:       "python-templates-finch",
		InternalJWTCoreAudience: "core-api",
	})
	_, err := client.NotificationEventsAfter(context.Background(), "t", "COGASA", "emilio@hawthornelabs.io", "", 100)
	if err != nil {
		t.Fatalf("NotificationEventsAfter: %v", err)
	}
	if !strings.Contains(gotQuery, "after=0") {
		t.Fatalf("python sends after=0 when Last-Event-ID is missing, got %q", gotQuery)
	}
}
