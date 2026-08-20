package cryptobffclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/security"
)

type CryptoBFFClient struct {
	baseURL    string
	httpClient *http.Client
	signer     *security.InternalJWTSigner
	audience   string
	cryptoVersion string
}

func NewCryptoBFFClient(cfg *config.Config) *CryptoBFFClient {
	timeout := time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &CryptoBFFClient{
		baseURL: cfg.CryptoBFFBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		signer: security.NewInternalJWTSigner(
			cfg.InternalJWTSecret,
			cfg.InternalJWTIssuer,
			cfg.InternalJWTActiveKID,
			5*time.Minute,
		),
		audience:      cfg.InternalJWTCryptoAudience,
		cryptoVersion: cfg.CryptoVersion,
	}
}

type encryptFieldsBody struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Fields map[string]string `json:"fields"`
}

type encryptFieldsResponse struct {
	Encrypted map[string]string `json:"encrypted"`
}

func (c *CryptoBFFClient) EncryptFields(
	ctx context.Context,
	method, path string,
	plaintext map[string]string,
	tenantID, requestID, correlationID, traceparent string,
) (map[string]string, error) {
	if len(plaintext) == 0 {
		return map[string]string{}, nil
	}
	body, err := json.Marshal(encryptFieldsBody{Method: method, Path: path, Fields: plaintext})
	if err != nil {
		return nil, fmt.Errorf("marshal encrypt-fields body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/crypto/encrypt-fields", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Crypto-Version", c.cryptoVersion)
	req.Header.Set("Crypto-Tenant-Id", tenantID)
	if token, err := c.signer.Mint(c.audience, "bff-api", ""); err == nil {
		req.Header.Set("Authorization", "Bearer "+token)
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crypto-bff encrypt-fields: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("crypto-bff encrypt-fields status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed encryptFieldsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Encrypted == nil {
		return map[string]string{}, nil
	}
	return parsed.Encrypted, nil
}

type decryptFieldsBody struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Fields map[string]string `json:"fields"`
}

type decryptFieldsResponse struct {
	Plaintext map[string]string `json:"plaintext"`
}

func (c *CryptoBFFClient) DecryptFields(
	ctx context.Context,
	method, path string,
	encrypted map[string]string,
	cryptoVersion, cryptoSessionID, cryptoRequestID, cryptoTimestamp, cryptoTenantID string,
	requestID, correlationID, traceparent string,
) (map[string]string, error) {
	if len(encrypted) == 0 {
		return map[string]string{}, nil
	}
	body, err := json.Marshal(decryptFieldsBody{Method: method, Path: path, Fields: encrypted})
	if err != nil {
		return nil, fmt.Errorf("marshal decrypt-fields body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/crypto/decrypt-fields", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Crypto-Version", cryptoVersion)
	req.Header.Set("Crypto-Session-Id", cryptoSessionID)
	req.Header.Set("Crypto-Request-Id", cryptoRequestID)
	req.Header.Set("Crypto-Timestamp", cryptoTimestamp)
	req.Header.Set("Crypto-Tenant-Id", cryptoTenantID)
	if token, err := c.signer.Mint(c.audience, "bff-api", ""); err == nil {
		req.Header.Set("Authorization", "Bearer "+token)
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crypto-bff decrypt-fields: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("crypto-bff decrypt-fields status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed decryptFieldsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Plaintext == nil {
		return map[string]string{}, nil
	}
	return parsed.Plaintext, nil
}

type EncryptRequest struct {
	CryptoVersion   string            `json:"crypto_version"`
	CryptoSessionID string            `json:"crypto_session_id"`
	CryptoRequestID string            `json:"crypto_request_id"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	CryptoTenantID  string            `json:"crypto_tenant_id"`
	PlaintextFields map[string]string `json:"plaintext_fields"`
}

type EncryptResponse struct {
	EncryptedFields map[string]string `json:"encrypted_fields"`
}

func (c *CryptoBFFClient) EncryptFieldsLegacy(ctx context.Context, encryptReq EncryptRequest) (*EncryptResponse, error) {
	encrypted, err := c.EncryptFields(
		ctx,
		encryptReq.Method,
		encryptReq.Path,
		encryptReq.PlaintextFields,
		encryptReq.CryptoTenantID,
		"",
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	return &EncryptResponse{EncryptedFields: encrypted}, nil
}

func (c *CryptoBFFClient) Handshake(ctx context.Context, body []byte, authHeader string) (int, []byte, error) {
	return c.HandshakeWithHeaders(ctx, body, "sp-fle-v1", "default", "", "", "", "")
}

func (c *CryptoBFFClient) HandshakeWithHeaders(
	ctx context.Context,
	body []byte,
	cryptoVersion, cryptoTenantID, traceID, requestID, correlationID, traceparent string,
) (int, []byte, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+"/api/v1/crypto/handshake",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("create handshake request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Crypto-Version", cryptoVersion)
	httpReq.Header.Set("Crypto-Tenant-Id", cryptoTenantID)
	if traceID != "" {
		httpReq.Header.Set("X-Trace-Id", traceID)
	}
	if requestID != "" {
		httpReq.Header.Set("X-Request-Id", requestID)
	}
	if correlationID != "" {
		httpReq.Header.Set("X-Correlation-Id", correlationID)
	}
	if traceparent != "" {
		httpReq.Header.Set("traceparent", traceparent)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("crypto-bff handshake: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read handshake response: %w", err)
	}
	return resp.StatusCode, respBody, nil
}

func (c *CryptoBFFClient) CreateSession(ctx context.Context, tenantID string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+"/api/v1/crypto-sessions", nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create session request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crypto-bff create session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("crypto-bff create session status %d: %s", resp.StatusCode, string(body))
	}
	var result map[string]any
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read session response: %w", err)
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse session response: %w", err)
	}
	return result, nil
}
