package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader     = "X-Request-Id"
	correlationIDHeader = "X-Correlation-Id"
	traceIDHeader       = "X-Trace-Id"
)

// RequestContextMiddleware resolves request/correlation ids and echoes them on responses.
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.New().String()
		}
		correlationID := strings.TrimSpace(c.GetHeader(correlationIDHeader))
		if correlationID == "" {
			correlationID = requestID
		}
		traceID := strings.TrimSpace(c.GetHeader(traceIDHeader))
		if traceID == "" {
			traceID = requestID
		}

		c.Set("request_id", requestID)
		c.Set("correlation_id", correlationID)
		c.Set("trace_id", traceID)

		c.Header(requestIDHeader, requestID)
		c.Header(correlationIDHeader, correlationID)

		c.Next()
	}
}
