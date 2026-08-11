package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/fieldcrypto"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api"
	activityhandler "github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/handler"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"

	sharedobservability "github.com/Hawthorne-Labs/shared-observability-go"
)

func main() {
	cfg := config.Load()

	// Initialize OpenTelemetry
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	shutdown, err := sharedobservability.Init(ctx, cfg.OTELServiceName)
	if err != nil {
		log.Fatalf("telemetry init failed: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Printf("telemetry shutdown error: %v", err)
		}
	}()

	// Initialize infrastructure clients
	coreClient := coreclient.NewCoreClient(cfg)
	cryptoClient := cryptobffclient.NewCryptoBFFClient(cfg)

	// Initialize usecases
	activitiesUC := usecases.NewActivitiesUsecase(coreClient, cryptoClient)
	escalationsUC := usecases.NewEscalationsUsecase(coreClient, cryptoClient)
	paymentPromisesUC := usecases.NewPaymentPromisesUsecase(coreClient, cryptoClient)
	agentPerformanceUC := usecases.NewAgentPerformanceUsecase(coreClient, cryptoClient)
	notificationsUC := usecases.NewNotificationsUsecase(coreClient, cryptoClient)
	dashboardUC := usecases.NewDashboardUsecase(coreClient, cryptoClient)

	// Initialize handlers
	activitiesHandler := activityhandler.NewActivitiesHandler(activitiesUC)
	escalationsHandler := activityhandler.NewEscalationsHandler(escalationsUC)
	paymentPromisesHandler := activityhandler.NewPaymentPromisesHandler(paymentPromisesUC)
	agentPerformanceHandler := activityhandler.NewAgentPerformanceHandler(agentPerformanceUC)
	notificationsHandler := activityhandler.NewNotificationsHandler(notificationsUC)
	dashboardHandler := activityhandler.NewDashboardHandler(dashboardUC)
	authHandler := activityhandler.NewAuthHandler()
	healthHandler := activityhandler.NewHealthHandler()
	cryptoSessionStore := fieldcrypto.NewSessionStore(cfg.CryptoSessionTTL)
	cryptoSessionMgr := fieldcrypto.NewSessionManager(cryptoSessionStore, cfg.CryptoSessionSecret, cfg.CryptoSessionIssuer, cfg.CryptoSessionTTL)
	cryptoSessionHandler := activityhandler.NewCryptoSessionHandler(cryptoSessionMgr)
	auditHandler := activityhandler.NewAuditHandler(coreClient)

	// Set Gin mode
	gin.SetMode(getEnvOrDefault("GIN_MODE", "release"))

	// Create Gin engine
	r := gin.New()
	r.Use(gin.Recovery())

	// Middleware pipeline
	r.Use(middleware.TracingMiddleware())
	r.Use(middleware.CognitoContextMiddleware())
	r.Use(middleware.AuditMiddleware())
	r.Use(middleware.CryptoMiddleware(cfg.CryptoEnabled, middleware.NewCryptoClient(cfg.CryptoBFFBaseURL)))
	r.Use(middleware.CORS(cfg.CORSSOrigins))

	// Register routes
	api.RegisterRoutes(r,
		activitiesHandler,
		escalationsHandler,
		paymentPromisesHandler,
		agentPerformanceHandler,
		notificationsHandler,
		dashboardHandler,
		authHandler,
		healthHandler,
		cryptoSessionHandler,
		auditHandler,
	)

	// Build server
	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
