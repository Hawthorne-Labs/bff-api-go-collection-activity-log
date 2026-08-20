package api

import (
	"github.com/gin-gonic/gin"
	activityhandler "github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/handler"
)

// RegisterRoutes registers all API routes on the Gin engine.
func RegisterRoutes(
	r *gin.Engine,
	activities *activityhandler.ActivitiesHandler,
	escalations *activityhandler.EscalationsHandler,
	paymentPromises *activityhandler.PaymentPromisesHandler,
	agentPerformance *activityhandler.AgentPerformanceHandler,
	notifications *activityhandler.NotificationsHandler,
	dashboard *activityhandler.DashboardHandler,
	auth *activityhandler.AuthHandler,
	health *activityhandler.HealthHandler,
	cryptoSession *activityhandler.CryptoSessionHandler,
	audit *activityhandler.AuditHandler,
	contacts *activityhandler.ContactsHandler,
) {
	// Health
	r.GET("/health", health.Check)
	r.GET("/health/live", health.Liveness)
	r.GET("/health/ready", health.Readiness)

	// Crypto session handshake (proxied to crypto-bff)
	r.POST("/api/v1/collections/crypto-session", cryptoSession.Handshake)

	// Auth routes
	r.GET("/api/v1/auth/login", auth.Login)
	r.GET("/api/v1/auth/callback", auth.Callback)
	r.POST("/api/v1/auth/logout", auth.Logout)
	r.GET("/api/v1/auth/me", auth.Me)
	r.POST("/api/v1/auth/dev-login", auth.DevLogin)

	// Activities routes
	r.GET("/api/v1/collections/activities", activities.ListActivities)
	r.POST("/api/v1/collections/activities", activities.CreateActivity)
	r.POST("/api/v1/collections/activities/batch", activities.CreateActivityBatch)
	r.GET("/api/v1/collections/loans/:loanId/activities", activities.ListActivities)
	r.POST("/api/v1/collections/loans/:loanId/activities", activities.CreateLoanActivity)

	// Escalations routes
	r.POST("/api/v1/collections/escalations", escalations.CreateEscalation)
	r.POST("/api/v1/collections/loans/:loanId/escalations", escalations.CreateLoanEscalation)
	r.GET("/api/v1/collections/escalations", escalations.ListEscalations)
	r.PATCH("/api/v1/collections/escalations/:id/status", escalations.UpdateEscalationStatus)
	r.POST("/api/v1/collections/escalations/:id/decisions", escalations.DecideEscalation)

	// Payment promises routes
	r.GET("/api/v1/collections/payment-promises", paymentPromises.ListPaymentPromises)

	// Agent performance routes
	r.GET("/api/v1/collections/agent-performance/kpis", agentPerformance.GetKPIs)
	r.GET("/api/v1/collections/agent-performance/goals", agentPerformance.GetGoals)
	r.GET("/api/v1/collections/agent-performance/ranking", agentPerformance.GetRanking)
	r.GET("/api/v1/collections/agent-performance/workload", agentPerformance.GetWorkload)
	r.GET("/api/v1/collections/agent-performance/report", agentPerformance.GetReport)

	// Dashboard routes
	r.GET("/api/v1/collections/dashboard/summary", dashboard.GetSummary)
	r.GET("/api/v1/collections/dashboard/alerts", dashboard.GetAlerts)

	// Notifications routes
	r.GET("/api/v1/notifications", notifications.ListNotifications)
	r.GET("/api/v1/notifications/stream", notifications.NotificationEventsAfter)
	r.GET("/api/v1/notifications/unread-count", notifications.UnreadCount)
	r.PUT("/api/v1/notifications/devices/current", notifications.RegisterDevice)
	r.DELETE("/api/v1/notifications/devices/current", notifications.RevokeDevice)
	r.GET("/api/v1/notifications/:id", notifications.GetDetail)
	r.POST("/api/v1/notifications/read-all", notifications.MarkAllRead)
	r.POST("/api/v1/notifications/:id/read", notifications.MarkRead)

	// M2M (machine-to-machine)
	r.GET("/api/m2m/whoami", activityhandler.M2MWhoami)

	// Audit routes
	r.GET("/api/v1/audit/recent", audit.Recent)
	r.GET("/api/v1/audit/events", audit.ByEntity)
	r.GET("/api/v1/audit/integrity", audit.Integrity)

	// Contacts
	r.POST("/api/v1/contacts", contacts.SubmitContact)
}
