package coreclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
)

func TestListActivitiesTargetsInternalActivitiesRoute(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	client := NewCoreClient(&config.Config{
		CoreBaseURL:             server.URL,
		RequestTimeoutSeconds:   5,
		InternalJWTSecret:         "dev-internal-jwt-secret-32-bytes-min",
		InternalJWTIssuer:       "python-templates-finch",
		InternalJWTCoreAudience: "core-api",
	})

	_, err := client.ListActivities(
		context.Background(),
		"trace-1",
		"COGASA",
		"",
		[]string{"loan-a", "loan-b"},
		"",
		"",
		"",
		"",
		"agent@example.com",
		20,
		0,
	)
	if err != nil {
		t.Fatalf("ListActivities returned error: %v", err)
	}
	if gotPath != "/internal/v1/activities" {
		t.Fatalf("expected /internal/v1/activities, got %q", gotPath)
	}
}

func TestCreateEscalationTargetsPublicCollectionsRoute(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"esc-1"}`))
	}))
	t.Cleanup(server.Close)

	client := NewCoreClient(&config.Config{
		CoreBaseURL:             server.URL,
		RequestTimeoutSeconds:   5,
		InternalJWTSecret:         "dev-internal-jwt-secret-32-bytes-min",
		InternalJWTIssuer:       "python-templates-finch",
		InternalJWTCoreAudience: "core-api",
	})

	_, err := client.CreateEscalation(
		context.Background(),
		map[string]any{"loan_id": "loan-1", "reason": "review"},
		"trace-1",
		"COGASA",
		"",
		"",
		"",
		"",
		"supervisor@example.com",
	)
	if err != nil {
		t.Fatalf("CreateEscalation returned error: %v", err)
	}
	if gotPath != "/api/v1/collections/escalations" {
		t.Fatalf("expected /api/v1/collections/escalations, got %q", gotPath)
	}
}

func TestAuthHeadersMintRealHS256JWT(t *testing.T) {
	client := NewCoreClient(&config.Config{
		CoreBaseURL:             "http://example.invalid",
		RequestTimeoutSeconds:   5,
		InternalJWTSecret:         "dev-internal-jwt-secret-32-bytes-min",
		InternalJWTIssuer:       "python-templates-finch",
		InternalJWTCoreAudience: "core-api",
	})

	headers, err := client.authHeaders(context.Background(), "trace-1", "COGASA", "Agent@Example.com")
	if err != nil {
		t.Fatalf("authHeaders returned error: %v", err)
	}
	auth := headers["Authorization"]
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Fatalf("expected bearer token, got %q", auth)
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected HS256 JWT with 3 segments, got %q", token)
	}
	if strings.HasPrefix(token, "INTERNAL.") {
		t.Fatal("legacy INTERNAL placeholder token must not be used")
	}
}
