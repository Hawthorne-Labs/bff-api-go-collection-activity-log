package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"
)

// PaymentPromisesHandler handles payment promise-related HTTP endpoints.
type PaymentPromisesHandler struct {
	paymentPromises *usecases.PaymentPromisesUsecase
}

// NewPaymentPromisesHandler creates a new PaymentPromisesHandler.
func NewPaymentPromisesHandler(paymentPromises *usecases.PaymentPromisesUsecase) *PaymentPromisesHandler {
	return &PaymentPromisesHandler{paymentPromises: paymentPromises}
}

// ListPaymentPromises handles GET /api/v1/collections/payment-promises
func (h *PaymentPromisesHandler) ListPaymentPromises(c *gin.Context) {
	ctx := middleware.GetCognitoContext(c)
	if ctx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": domain.InvalidAuthToken, "message": "El token de acceso no es válido."}})
		return
	}

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 500 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	loanID := c.Query("loan_id")
	clientID := c.Query("client_id")
	agentID := c.Query("agent_id")
	agentName := c.Query("agent_name")

	result, err := h.paymentPromises.ListPaymentPromises(c.Request.Context(), loanID, clientID, agentID, agentName, limit, offset, traceID.(string), tenantID.(string), ctx.Email)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": bizErr.Code, "message": bizErr.Message}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": map[string]any{"code": domain.PaymentPromisesListFailed, "message": "No se pudo cargar la lista de promesas de pago."}})
		return
	}

	c.JSON(http.StatusOK, result)
}
