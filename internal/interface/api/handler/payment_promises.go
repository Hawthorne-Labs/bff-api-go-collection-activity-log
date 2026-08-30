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
	if _, ok := middleware.RequireScope(c, "collections:read"); !ok {
		return
	}
	if _, ok := middleware.EnforceAllRoles(c); !ok {
		return
	}
	ctx := middleware.GetCognitoContext(c)

	traceID, _ := c.Get("trace_id")
	tenantID, _ := c.Get("tenant_id")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 || limit > 500 {
		limit = 50
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
		writeErr(c, err, domain.PaymentPromisesListFailed, "No se pudo cargar la lista de promesas de pago.")
		return
	}

	c.JSON(http.StatusOK, result)
}
