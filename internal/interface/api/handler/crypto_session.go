package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/fieldcrypto"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// CryptoSessionHandler handles the P-256 ECDH crypto-session handshake.
type CryptoSessionHandler struct {
	mgr             any
	tenantAuthority fieldcrypto.TenantAuthority
}

// NewCryptoSessionHandler creates a new handler.
func NewCryptoSessionHandler(mgr any, tenantAuthority fieldcrypto.TenantAuthority) *CryptoSessionHandler {
	if tenantAuthority == nil {
		tenantAuthority = fieldcrypto.GetTenantAuthority()
	}
	return &CryptoSessionHandler{mgr: mgr, tenantAuthority: tenantAuthority}
}

type handshakeRequest struct {
	ClientPublicKey string `json:"clientPublicKey"`
}

// Handshake handles POST /api/v1/collections/crypto-session
func (h *CryptoSessionHandler) Handshake(c *gin.Context) {
	var req handshakeRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil || req.ClientPublicKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": 90100, "message": "Solicitud de cifrado inválida."}})
		return
	}

	sub := "user"
	scope := "collections:read"
	email := ""
	if cc := middleware.GetCognitoContext(c); cc != nil {
		if cc.Sub != "" {
			sub = cc.Sub
		}
		if cc.Scope != "" {
			scope = cc.Scope
		}
		email = cc.Email
	}

	switch mgr := h.mgr.(type) {
	case *fieldcrypto.StatelessCryptoSessionManager:
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": 90109, "message": fieldcrypto.CatalogErrorMessage(90109)}})
			return
		}
		authorized, err := h.tenantAuthority.Resolve(
			c.Request.Context(),
			authorization,
			c.GetHeader("X-Tenant-Id"),
			email,
			true,
			c.GetHeader("X-Trace-Id"),
		)
		if err != nil {
			status, body := fieldcrypto.PublicErrorEnvelope(err)
			c.JSON(status, body)
			return
		}
		accessTokenHash, err := fieldcrypto.HashAccessToken(authorization)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": 90109, "message": fieldcrypto.CatalogErrorMessage(90109)}})
			return
		}
		result, err := mgr.Handshake(req.ClientPublicKey, sub, scope, authorized.TenantDigest, accessTokenHash)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": 90101, "message": fieldcrypto.CatalogErrorMessage(90101)}})
			return
		}
		c.JSON(http.StatusOK, result)
	case *fieldcrypto.CryptoSessionManager:
		result, err := mgr.Handshake(req.ClientPublicKey, sub, scope)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": 90101, "message": fieldcrypto.CatalogErrorMessage(90101)}})
			return
		}
		c.JSON(http.StatusOK, result)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 90012, "message": "Error interno."}})
	}
}
