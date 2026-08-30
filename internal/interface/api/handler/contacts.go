package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/fle"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

const (
	contactMaxName    = 120
	contactMaxEmail   = 254
	contactMaxMessage = 4000
	contactsRoutePath = "/api/v1/contacts"
)

// ContactsHandler handles contact-related HTTP endpoints.
type ContactsHandler struct {
	contacts *usecases.ContactsUsecase
	crypto   *cryptobffclient.CryptoBFFClient
}

// NewContactsHandler creates a new ContactsHandler.
func NewContactsHandler(contacts *usecases.ContactsUsecase, crypto *cryptobffclient.CryptoBFFClient) *ContactsHandler {
	return &ContactsHandler{contacts: contacts, crypto: crypto}
}

// SubmitContact handles POST /api/v1/contacts
func (h *ContactsHandler) SubmitContact(c *gin.Context) {
	if middleware.SessionPayload(c) == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "no session"})
		return
	}

	policy, ok := fle.PolicyFor(http.MethodPost, contactsRoutePath)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "no encryption policy for route"})
		return
	}

	cryptoVersion := c.GetHeader("Crypto-Version")
	cryptoSessionID := c.GetHeader("Crypto-Session-Id")
	cryptoRequestID := c.GetHeader("Crypto-Request-Id")
	cryptoTimestamp := c.GetHeader("Crypto-Timestamp")
	cryptoTenantID := c.GetHeader("Crypto-Tenant-Id")
	if cryptoVersion == "" || cryptoSessionID == "" || cryptoRequestID == "" || cryptoTimestamp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "missing crypto headers"})
		return
	}
	if cryptoTenantID == "" {
		cryptoTenantID = "default"
	}

	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "body is not valid json"})
		return
	}

	plaintext, err := fle.DecryptRequestBody(body, policy, func(fields map[string]string) (map[string]string, error) {
		return h.crypto.DecryptFields(
			c.Request.Context(),
			http.MethodPost,
			contactsRoutePath,
			fields,
			cryptoVersion,
			cryptoSessionID,
			cryptoRequestID,
			cryptoTimestamp,
			cryptoTenantID,
			c.GetString("request_id"),
			c.GetString("correlation_id"),
			c.GetHeader("traceparent"),
		)
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	name, _ := plaintext["name"].(string)
	email, _ := plaintext["email"].(string)
	message, _ := plaintext["message"].(string)
	if name == "" || email == "" || message == "" {
		writeAPIError(c, http.StatusUnprocessableEntity, domain.ValidationError, "Los campos name, email y message son obligatorios.")
		return
	}
	if len(name) > contactMaxName {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"detail": "name exceeds 120 chars"})
		return
	}
	if len(email) > contactMaxEmail {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"detail": "email exceeds 254 chars"})
		return
	}
	if len(message) > contactMaxMessage {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"detail": "message exceeds 4000 chars"})
		return
	}

	traceID, _ := c.Get("trace_id")
	effectiveTraceID := traceID.(string)
	if headerTrace := c.GetHeader("X-Trace-Id"); headerTrace != "" {
		effectiveTraceID = headerTrace
	} else if cryptoRequestID != "" {
		effectiveTraceID = "trace-" + cryptoRequestID
	}

	result, err := h.contacts.SubmitContact(
		c.Request.Context(),
		plaintext,
		effectiveTraceID,
		cryptoTenantID,
		c.GetString("request_id"),
		c.GetString("correlation_id"),
		c.GetHeader("traceparent"),
	)
	if err != nil {
		writeErr(c, err, domain.ContactSubmitFailed, "No se pudo enviar la solicitud de contacto.")
		return
	}

	c.JSON(http.StatusCreated, result)
}
