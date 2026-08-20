package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

const (
	contactMaxName    = 120
	contactMaxEmail   = 254
	contactMaxMessage = 4000
)

// ContactsHandler handles contact-related HTTP endpoints.
type ContactsHandler struct {
	contacts *usecases.ContactsUsecase
}

// NewContactsHandler creates a new ContactsHandler.
func NewContactsHandler(contacts *usecases.ContactsUsecase) *ContactsHandler {
	return &ContactsHandler{contacts: contacts}
}

type submitContactRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// SubmitContact handles POST /api/v1/contacts
func (h *ContactsHandler) SubmitContact(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	var req submitContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ContactSubmitFailed, "message": "Los campos name, email y message son obligatorios."}})
		return
	}

	if len(req.Name) > contactMaxName {
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ContactSubmitFailed, "message": "El nombre no puede exceder 120 caracteres."}})
		return
	}
	if len(req.Email) > contactMaxEmail {
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ContactSubmitFailed, "message": "El email no puede exceder 254 caracteres."}})
		return
	}
	if len(req.Message) > contactMaxMessage {
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ContactSubmitFailed, "message": "El mensaje no puede exceder 4000 caracteres."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")
	requestID, _ := c.Get("crypto_request_id")
	cryptoTenantID, _ := c.Get("crypto_tenant_id")

	effectiveTenant := tenantID.(string)
	if ct, ok := cryptoTenantID.(string); ok && ct != "" {
		effectiveTenant = ct
	}

	payload := map[string]any{
		"name":    req.Name,
		"email":   req.Email,
		"message": req.Message,
	}

	result, err := h.contacts.SubmitContact(
		c.Request.Context(),
		payload,
		traceID.(string),
		effectiveTenant,
		requestID.(string),
		"",
		c.GetHeader("traceparent"),
	)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.ContactSubmitFailed, "message": "No se pudo enviar el formulario de contacto."}})
		return
	}

	c.JSON(http.StatusCreated, result)
}
