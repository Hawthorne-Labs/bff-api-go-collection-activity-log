package coreclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/security"
)

// CoreClient is an HTTP client for the collections-operations core.
// It owns an http.Client with connection pooling and an internal JWT signer.
type CoreClient struct {
	baseURL    string
	httpClient *http.Client
	jwtSigner  *security.InternalJWTSigner
	audience   string
	subject    string
}

// mtlsBundle is the JSON structure from MTLS_BUNDLE_JSON secret.
type mtlsBundle struct {
	CertificatePEM string `json:"certificate_pem"`
	PrivateKeyPEM  string `json:"private_key_pem"`
	TrustBundlePEM string `json:"trust_bundle_pem"`
}

// loadMtlsTransport builds an HTTP transport with mTLS client identity
// from the MTLS_BUNDLE_JSON env var. Falls back to plain transport if
// the env var is not set.
func loadMtlsTransport() *http.Transport {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	bundleJSON := os.Getenv("MTLS_BUNDLE_JSON")
	if bundleJSON == "" {
		log.Println("mTLS: MTLS_BUNDLE_JSON not set, using plain transport")
		return transport
	}

	var bundle mtlsBundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		log.Printf("mTLS: failed to parse MTLS_BUNDLE_JSON: %v, using plain transport", err)
		return transport
	}

	cert, err := tls.X509KeyPair([]byte(bundle.CertificatePEM), []byte(bundle.PrivateKeyPEM))
	if err != nil {
		log.Printf("mTLS: failed to load client cert/key: %v, using plain transport", err)
		return transport
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(bundle.TrustBundlePEM)) {
		log.Println("mTLS: failed to parse trust bundle, using plain transport")
		return transport
	}

	transport.TLSClientConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}
	log.Println("mTLS: client identity loaded from MTLS_BUNDLE_JSON")
	return transport
}

// NewMtlsHTTPClient returns an HTTP client using the task mTLS bundle when present.
func NewMtlsHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &http.Client{
		Transport: loadMtlsTransport(),
		Timeout:   timeout,
	}
}

func NewCoreClient(cfg *config.Config) *CoreClient {
	client := NewMtlsHTTPClient(time.Duration(cfg.RequestTimeoutSeconds) * time.Second)

	secret := cfg.InternalJWTSecret
	issuer := cfg.InternalJWTIssuer
	audience := cfg.InternalJWTCoreAudience
	signer := security.NewInternalJWTSigner(secret, issuer, cfg.InternalJWTActiveKID, 5*time.Minute)

	return &CoreClient{
		baseURL:    cfg.CoreBaseURL,
		httpClient: client,
		jwtSigner:  signer,
		audience:   audience,
		subject:    "bff-api",
	}
}

// GetToken mints a new internal JWT without actor scope.
func (c *CoreClient) GetToken(_ context.Context) (string, error) {
	return c.jwtSigner.Mint(c.audience, c.subject, "")
}

// authHeaders returns the standard Authorization + tracing headers for core requests.
func (c *CoreClient) authHeaders(_ context.Context, traceID, tenantID, userEmail string) (map[string]string, error) {
	token, err := c.jwtSigner.Mint(c.audience, c.subject, userEmail)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"Authorization": "Bearer " + token,
		"X-Trace-Id":    traceID,
		"X-Tenant-Id":   tenantID,
	}, nil
}

// get performs an HTTP GET to the core and returns the parsed JSON response.
func (c *CoreClient) get(ctx context.Context, path string, headers map[string]string, params map[string]string) (map[string]any, error) {
	endpoint := c.baseURL + path
	query := ""
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		query = "?" + values.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+query, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, translateCoreError(resp.StatusCode, body)
	}

	var result map[string]any
	if len(body) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}

// ForwardGet performs a raw GET to the core, returning the HTTP response.
// Used for proxy endpoints (audit, etc.) where the body is forwarded as-is.
func (c *CoreClient) ForwardGet(ctx context.Context, path string, params map[string]string, traceID, requestID, correlationID, traceparent string) (*http.Response, error) {
	url := c.baseURL + path
	if len(params) > 0 {
		queryParts := make([]string, 0, len(params))
		for k, v := range params {
			queryParts = append(queryParts, k+"="+v)
		}
		url += "?" + strings.Join(queryParts, "&")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	token, _ := c.jwtSigner.Mint(c.audience, c.subject, "")
	req.Header.Set("Authorization", "Bearer "+token)
	if traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if correlationID != "" {
		req.Header.Set("X-Correlation-Id", correlationID)
	}
	if traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}
	return c.httpClient.Do(req)
}

// post performs an HTTP POST to the core and returns the parsed JSON response.
func (c *CoreClient) post(ctx context.Context, path string, headers map[string]string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, translateCoreError(resp.StatusCode, respBody)
	}

	var result map[string]any
	if len(respBody) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}

// patch performs an HTTP PATCH to the core and returns the parsed JSON response.
func (c *CoreClient) patch(ctx context.Context, path string, headers map[string]string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, translateCoreError(resp.StatusCode, respBody)
	}

	var result map[string]any
	if len(respBody) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}

// put performs an HTTP PUT to the core and returns the parsed JSON response.
func (c *CoreClient) put(ctx context.Context, path string, headers map[string]string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core PUT %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, translateCoreError(resp.StatusCode, respBody)
	}

	var result map[string]any
	if len(respBody) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return result, nil
}

// deleteNoContent performs an HTTP DELETE that expects 204 No Content.
func (c *CoreClient) deleteNoContent(ctx context.Context, path string, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("core DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return translateCoreError(resp.StatusCode, respBody)
	}
	return nil
}

// postNoContent performs an HTTP POST that expects 204 No Content.
func (c *CoreClient) postNoContent(ctx context.Context, path string, headers map[string]string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("core POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return translateCoreError(resp.StatusCode, respBody)
	}
	return nil
}

// ------------------------------------------------------------------
// Public API methods — mirror the Python CoreClient
// ------------------------------------------------------------------

// ListActivities lists activities with filtering by trace_id, tenant_id, loan_id, loan_ids, client_id, agent_id, activity_type, etc.
func (c *CoreClient) ListActivities(ctx context.Context, traceID, tenantID, loanID string, loanIDs []string, clientID, agentID, agentName, activityType, userEmail string, limit, offset int) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"limit":  fmt.Sprintf("%d", limit),
		"offset": fmt.Sprintf("%d", offset),
	}
	if loanID != "" {
		params["loan_id"] = loanID
	}
	if len(loanIDs) > 0 {
		params["loan_ids"] = strings.Join(loanIDs, ",")
	}
	if clientID != "" {
		params["client_id"] = clientID
	}
	if agentID != "" {
		params["agent_id"] = agentID
	}
	if agentName != "" {
		params["agent_name"] = agentName
	}
	if activityType != "" {
		params["activity_type"] = activityType
	}
	return c.get(ctx, "/internal/v1/activities", headers, params)
}

// CreateActivity creates a single activity with idempotency support.
func (c *CoreClient) CreateActivity(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	if requestID != "" {
		headers["X-Request-Id"] = requestID
	}
	if correlationID != "" {
		headers["X-Correlation-Id"] = correlationID
	}
	if traceparent != "" {
		headers["traceparent"] = traceparent
	}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	return c.post(ctx, "/internal/v1/activities", headers, payload)
}

// CreateActivityBatch creates activities in batch.
func (c *CoreClient) CreateActivityBatch(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	if requestID != "" {
		headers["X-Request-Id"] = requestID
	}
	if correlationID != "" {
		headers["X-Correlation-Id"] = correlationID
	}
	if traceparent != "" {
		headers["traceparent"] = traceparent
	}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	return c.post(ctx, "/internal/v1/activities/batch", headers, payload)
}

// CreateEscalation creates an escalation record.
func (c *CoreClient) CreateEscalation(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent, idempotencyKey, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	if requestID != "" {
		headers["X-Request-Id"] = requestID
	}
	if correlationID != "" {
		headers["X-Correlation-Id"] = correlationID
	}
	if traceparent != "" {
		headers["traceparent"] = traceparent
	}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	return c.post(ctx, "/api/v1/collections/escalations", headers, payload)
}

// UpdateEscalationStatus updates the status of an escalation.
func (c *CoreClient) UpdateEscalationStatus(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.patch(ctx, "/api/v1/collections/escalations/"+escalationID+"/status", headers, payload)
}

// DecideEscalation records a decision on an escalation.
func (c *CoreClient) DecideEscalation(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.post(ctx, "/api/v1/collections/escalations/"+escalationID+"/decisions", headers, payload)
}

// ListEscalations lists escalations (uses activity_type=escalation filter internally).
func (c *CoreClient) ListEscalations(ctx context.Context, traceID, tenantID, loanID, clientID, agentID, agentName, status, userEmail string, limit, offset int) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"activity_type": "escalation",
		"limit":         fmt.Sprintf("%d", limit),
		"offset":        fmt.Sprintf("%d", offset),
	}
	if loanID != "" {
		params["loan_id"] = loanID
	}
	if clientID != "" {
		params["client_id"] = clientID
	}
	if agentID != "" {
		params["agent_id"] = agentID
	}
	if agentName != "" {
		params["agent_name"] = agentName
	}
	if status != "" {
		params["status"] = status
	}
	return c.get(ctx, "/internal/v1/activities", headers, params)
}

// GetAgentKPIs gets agent KPIs for a given day.
func (c *CoreClient) GetAgentKPIs(ctx context.Context, agentID, traceID, tenantID, day string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, "")
	if err != nil {
		return nil, err
	}
	params := map[string]string{}
	if day != "" {
		params["day"] = day
	}
	return c.get(ctx, "/internal/v1/kpis/agents/"+agentID, headers, params)
}

// GetAgentGoals gets agent goals.
func (c *CoreClient) GetAgentGoals(ctx context.Context, agentID, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/kpis/agents/"+agentID+"/goals", headers, nil)
}

// GetRanking gets the agent performance ranking.
func (c *CoreClient) GetRanking(ctx context.Context, traceID, tenantID, day string, limit int, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}
	if day != "" {
		params["day"] = day
	}
	return c.get(ctx, "/internal/v1/kpis/ranking", headers, params)
}

// GetWorkload gets agent workload.
func (c *CoreClient) GetWorkload(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/kpis/workload", headers, nil)
}

// GetTeamPerformanceReport gets the team performance report.
func (c *CoreClient) GetTeamPerformanceReport(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/kpis/team-performance-report", headers, nil)
}

// GetScopedGoals gets scoped goals for the current user/team.
func (c *CoreClient) GetScopedGoals(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/kpis/goals", headers, nil)
}

// GetOperationsSummary gets the admin operations summary for a tenant/day.
func (c *CoreClient) GetOperationsSummary(ctx context.Context, traceID, tenantID, userEmail, tenant, day string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{"tenant": tenant}
	if day != "" {
		params["date"] = day
	}
	return c.get(ctx, "/internal/v1/kpis/operations-summary", headers, params)
}

// ListNotifications lists notifications with cursor pagination and filters.
func (c *CoreClient) ListNotifications(ctx context.Context, traceID, tenantID, userEmail, state, severity, fromDate, toDate string, limit int, beforeAt, beforeID string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}
	if state != "" {
		params["state"] = state
	}
	if severity != "" {
		params["severity"] = severity
	}
	if fromDate != "" {
		params["from"] = fromDate
	}
	if toDate != "" {
		params["to"] = toDate
	}
	if beforeAt != "" && beforeID != "" {
		params["before_at"] = beforeAt
		params["before_id"] = beforeID
	}
	return c.get(ctx, "/internal/v1/notifications", headers, params)
}

// NotificationEventsAfter gets notification events after a given event ID (for SSE).
func (c *CoreClient) NotificationEventsAfter(ctx context.Context, traceID, tenantID, userEmail, afterEventID string, limit int) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"limit": fmt.Sprintf("%d", limit),
	}
	if afterEventID != "" {
		params["after"] = afterEventID
	} else {
		params["after"] = "0"
	}
	return c.get(ctx, "/internal/v1/notifications/events", headers, params)
}

// NotificationUnreadCount gets the unread notification count.
func (c *CoreClient) NotificationUnreadCount(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/notifications/unread-count", headers, nil)
}

// RegisterNotificationDevice registers a notification device.
func (c *CoreClient) RegisterNotificationDevice(ctx context.Context, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.put(ctx, "/internal/v1/notifications/devices/current", headers, payload)
}

// RevokeNotificationDevice revokes a notification device.
func (c *CoreClient) RevokeNotificationDevice(ctx context.Context, installationID, traceID, tenantID, userEmail string) error {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return err
	}
	path := "/internal/v1/notifications/devices/current"
	if installationID != "" {
		path += "?installation_id=" + url.QueryEscape(installationID)
	}
	return c.deleteNoContent(ctx, path, headers)
}

// NotificationDetail gets a single notification detail.
func (c *CoreClient) NotificationDetail(ctx context.Context, notificationID, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/notifications/"+notificationID, headers, nil)
}

// MarkAllNotificationsRead marks all notifications as read.
func (c *CoreClient) MarkAllNotificationsRead(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.post(ctx, "/internal/v1/notifications/read-all", headers, map[string]any{})
}

// MarkNotificationRead marks a single notification as read.
func (c *CoreClient) MarkNotificationRead(ctx context.Context, notificationID, traceID, tenantID, userEmail string) error {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/internal/v1/notifications/"+notificationID+"/read", headers, nil)
}

// SubmitContact submits a contact form to the core.
func (c *CoreClient) SubmitContact(ctx context.Context, payload map[string]any, traceID, tenantID, requestID, correlationID, traceparent string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, "")
	if err != nil {
		return nil, err
	}
	if requestID != "" {
		headers["X-Request-Id"] = requestID
	}
	if correlationID != "" {
		headers["X-Correlation-Id"] = correlationID
	}
	if traceparent != "" {
		headers["traceparent"] = traceparent
	}
	return c.post(ctx, "/internal/v1/contacts", headers, payload)
}
