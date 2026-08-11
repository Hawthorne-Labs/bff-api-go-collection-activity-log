package coreclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
)

// CoreClient is an HTTP client for the collections-operations core.
// It owns an http.Client with connection pooling and an internal JWT signer.
type CoreClient struct {
	baseURL    string
	httpClient *http.Client
	jwtSigner  *JWTSigner
}

// JWTSigner mints internal JWTs for BFF→Core auth.
type JWTSigner struct {
	issuer   string
	audience string
	secret   string
	ttl      time.Duration
}

// NewJWTSigner creates a JWT signer using HS256 with the given secret.
func NewJWTSigner(issuer, audience, secret string) *JWTSigner {
	return &JWTSigner{
		issuer:   issuer,
		audience: audience,
		secret:   secret,
		ttl:      5 * time.Minute,
	}
}

// Mint creates a new internal JWT token.
func (s *JWTSigner) Mint(ctx context.Context) (string, error) {
	iat := time.Now().Unix()
	exp := iat + int64(s.ttl.Seconds())
	payload := map[string]any{
		"iss": s.issuer,
		"aud": s.audience,
		"iat": iat,
		"exp": exp,
		"sub": "bff-api",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("INTERNAL.%s", base64.URLEncoding.EncodeToString(data))
	return token, nil
}

// NewCoreClient creates a new CoreClient with the given config.
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

func NewCoreClient(cfg *config.Config) *CoreClient {
	transport := loadMtlsTransport()

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.RequestTimeoutSeconds) * time.Second,
	}

	// JWT signer uses CORE_JWT_SECRET env var or default
	secret := os.Getenv("CORE_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	signer := NewJWTSigner("bff-api", "core-operations", secret)

	return &CoreClient{
		baseURL:    cfg.CoreBaseURL,
		httpClient: client,
		jwtSigner:  signer,
	}
}

// GetToken mints a new internal JWT.
func (c *CoreClient) GetToken(ctx context.Context) (string, error) {
	return c.jwtSigner.Mint(ctx)
}

// authHeaders returns the standard Authorization + tracing headers for core requests.
func (c *CoreClient) authHeaders(ctx context.Context, traceID, tenantID, userEmail string) (map[string]string, error) {
	token, err := c.jwtSigner.Mint(ctx)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"X-Trace-Id":    traceID,
		"X-Tenant-Id":   tenantID,
	}
	if userEmail != "" {
		headers["X-User-Email"] = userEmail
	}
	return headers, nil
}

// get performs an HTTP GET to the core and returns the parsed JSON response.
func (c *CoreClient) get(ctx context.Context, path string, headers map[string]string, params map[string]string) (map[string]any, error) {
	url := c.baseURL + path
	query := ""
	if len(params) > 0 {
		queryParts := make([]string, 0, len(params))
		for k, v := range params {
			queryParts = append(queryParts, k+"="+v)
		}
		query = "?" + strings.Join(queryParts, "&")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+query, nil)
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
	token, _ := c.jwtSigner.Mint(ctx)
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

// ListActivities lists activities with filtering by trace_id, tenant_id, loan_id, client_id, agent_id, activity_type, etc.
func (c *CoreClient) ListActivities(ctx context.Context, traceID, tenantID, loanID, clientID, agentID, agentName, activityType, userEmail string, limit, offset int) (map[string]any, error) {
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
	return c.get(ctx, "/internal/v1/collections/activities", headers, params)
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
	return c.post(ctx, "/internal/v1/collections/activities", headers, payload)
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
	return c.post(ctx, "/internal/v1/collections/activities/batch", headers, payload)
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
	return c.post(ctx, "/internal/v1/collections/escalations", headers, payload)
}

// UpdateEscalationStatus updates the status of an escalation.
func (c *CoreClient) UpdateEscalationStatus(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.patch(ctx, "/internal/v1/collections/escalations/"+escalationID+"/status", headers, payload)
}

// DecideEscalation records a decision on an escalation.
func (c *CoreClient) DecideEscalation(ctx context.Context, escalationID string, payload map[string]any, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.post(ctx, "/internal/v1/collections/escalations/"+escalationID+"/decisions", headers, payload)
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
	return c.get(ctx, "/internal/v1/collections/escalations", headers, params)
}

// GetAgentKPIs gets agent KPIs for a given day.
func (c *CoreClient) GetAgentKPIs(ctx context.Context, agentID, traceID, tenantID, day string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, "")
	if err != nil {
		return nil, err
	}
	params := map[string]string{}
	if agentID != "" {
		params["agent_id"] = agentID
	}
	if day != "" {
		params["day"] = day
	}
	return c.get(ctx, "/internal/v1/collections/agent-performance/kpis", headers, params)
}

// GetAgentGoals gets agent goals.
func (c *CoreClient) GetAgentGoals(ctx context.Context, agentID, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	params := map[string]string{}
	if agentID != "" {
		params["agent_id"] = agentID
	}
	return c.get(ctx, "/internal/v1/collections/agent-performance/goals", headers, params)
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
	return c.get(ctx, "/internal/v1/collections/agent-performance/ranking", headers, params)
}

// GetWorkload gets agent workload.
func (c *CoreClient) GetWorkload(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/collections/agent-performance/workload", headers, nil)
}

// GetTeamPerformanceReport gets the team performance report.
func (c *CoreClient) GetTeamPerformanceReport(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/collections/agent-performance/report", headers, nil)
}

// GetScopedGoals gets scoped goals for the current user/team.
func (c *CoreClient) GetScopedGoals(ctx context.Context, traceID, tenantID, userEmail string) (map[string]any, error) {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	return c.get(ctx, "/internal/v1/collections/agent-performance/goals/scoped", headers, nil)
}

// ListNotifications lists notifications with cursor pagination and filters.
func (c *CoreClient) ListNotifications(ctx context.Context, traceID, tenantID, userEmail, state, severity, fromDate, toDate string, limit int, beforeID string) (map[string]any, error) {
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
		params["from_date"] = fromDate
	}
	if toDate != "" {
		params["to_date"] = toDate
	}
	if beforeID != "" {
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
		params["after_event_id"] = afterEventID
	}
	return c.get(ctx, "/internal/v1/notifications/stream", headers, params)
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
	return c.post(ctx, "/internal/v1/notifications/devices/current", headers, payload)
}

// RevokeNotificationDevice revokes a notification device.
func (c *CoreClient) RevokeNotificationDevice(ctx context.Context, installationID, traceID, tenantID, userEmail string) error {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/internal/v1/notifications/devices/current", headers, map[string]any{
		"installation_id": installationID,
	})
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
func (c *CoreClient) MarkAllNotificationsRead(ctx context.Context, traceID, tenantID, userEmail string) error {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/internal/v1/notifications/read-all", headers, nil)
}

// MarkNotificationRead marks a single notification as read.
func (c *CoreClient) MarkNotificationRead(ctx context.Context, notificationID, traceID, tenantID, userEmail string) error {
	headers, err := c.authHeaders(ctx, traceID, tenantID, userEmail)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/internal/v1/notifications/"+notificationID+"/read", headers, nil)
}
