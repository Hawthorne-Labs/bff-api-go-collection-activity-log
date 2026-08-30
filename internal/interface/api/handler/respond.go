package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
)

func writeAPIError(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, gin.H{"error": map[string]any{"code": code, "message": message}})
}

func writeErr(c *gin.Context, err error, fallbackCode int, fallbackMessage string) {
	if bizErr, ok := err.(*domain.BusinessError); ok {
		status := bizErr.Status()
		if status == http.StatusOK {
			status = http.StatusBadGateway
		}
		writeAPIError(c, status, bizErr.Code, bizErr.Message)
		return
	}
	writeAPIError(c, http.StatusBadGateway, fallbackCode, fallbackMessage)
}
