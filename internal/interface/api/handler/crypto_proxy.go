package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
)

// CryptoProxyHandler proxies crypto-bff handshake for cookie-authenticated clients.
type CryptoProxyHandler struct {
	crypto *cryptobffclient.CryptoBFFClient
}

// NewCryptoProxyHandler creates a CryptoProxyHandler.
func NewCryptoProxyHandler(crypto *cryptobffclient.CryptoBFFClient) *CryptoProxyHandler {
	return &CryptoProxyHandler{crypto: crypto}
}

// ProxyHandshake forwards POST /api/v1/crypto/handshake to crypto-bff.
func (h *CryptoProxyHandler) ProxyHandshake(c *gin.Context) {
	cryptoVersion := c.GetHeader("Crypto-Version")
	if cryptoVersion == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Crypto-Version header required"})
		return
	}
	cryptoTenantID := c.GetHeader("Crypto-Tenant-Id")
	if cryptoTenantID == "" {
		cryptoTenantID = "default"
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid request body"})
		return
	}

	statusCode, respBody, err := h.crypto.HandshakeWithHeaders(
		c.Request.Context(),
		body,
		cryptoVersion,
		cryptoTenantID,
		c.GetHeader("X-Trace-Id"),
		c.GetString("request_id"),
		c.GetString("correlation_id"),
		c.GetHeader("traceparent"),
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"detail": "crypto-bff unreachable: " + err.Error()})
		return
	}
	if statusCode >= 400 {
		c.Data(statusCode, "application/json", respBody)
		return
	}
	c.Data(http.StatusOK, "application/json", respBody)
}
