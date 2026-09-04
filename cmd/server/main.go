package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/application/usecases"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/auth"
	keycloakauth "github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/auth/keycloak"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/coreclient"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/cryptobffclient"
	activitiesinfra "github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/activities"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/fieldcrypto"
	notificationsinfra "github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/notifications"
	redisinfra "github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/redis"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/session"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api"
	activityhandler "github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/handler"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/interface/api/middleware"

	sharedobservability "github.com/Hawthorne-Labs/shared-observability-go"
)

func main() {
	cfg := config.Load()
	if err := config.EnforceProductionSecrets(); err != nil {
		log.Fatalf("%v", err)
	}

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

	emailLookup, err := auth.NewAWSCognitoEmailLookup(ctx, cfg.AWSRegion, cfg.CognitoPoolID)
	if err != nil {
		log.Printf("cognito AdminGetUser client unavailable: %v", err)
	}
	cognitoValidator := auth.NewCognitoJwtValidator(cfg, emailLookup, auth.NewIdentityEmailCache(cfg.RedisURL))
	redisClient := redisinfra.NewClient(cfg.RedisURL)
	var sessionStore *session.Store
	var flowStateStore *session.FlowStateStore
	if strings.EqualFold(cfg.SessionBackend, "memory") {
		sessionStore = session.NewMemoryStore()
		flowStateStore = session.NewMemoryFlowStateStore()
	} else {
		sessionStore = session.NewStore(redisClient)
		flowStateStore = session.NewFlowStateStore(redisClient)
	}
	keycloakPublicURL := cfg.KeycloakPublicURL
	if keycloakPublicURL == "" {
		keycloakPublicURL = cfg.KeycloakURL
	}
	oidcClient := keycloakauth.NewKeycloakOIDCClient(
		cfg.KeycloakURL,
		keycloakPublicURL,
		cfg.KeycloakRealm,
		cfg.KeycloakClientID,
		cfg.KeycloakTimeoutSeconds,
	)

	// Initialize infrastructure clients
	coreClient := coreclient.NewCoreClient(cfg)
	cryptoClient := cryptobffclient.NewCryptoBFFClient(cfg)

	// Initialize usecases
	activityReadCache := activitiesinfra.NewReadCache(redisClient)
	activitiesUC := usecases.NewActivitiesUsecase(coreClient, cryptoClient, activityReadCache)
	escalationsUC := usecases.NewEscalationsUsecase(coreClient, cryptoClient)
	paymentPromisesUC := usecases.NewPaymentPromisesUsecase(coreClient, cryptoClient)
	agentPerformanceUC := usecases.NewAgentPerformanceUsecase(coreClient, cryptoClient)
	notificationReadCache := notificationsinfra.NewReadCache(redisClient)
	notificationsUC := usecases.NewNotificationsUsecase(coreClient, cryptoClient, notificationReadCache)
	dashboardUC := usecases.NewDashboardUsecase(coreClient, cryptoClient)
	contactsUC := usecases.NewContactsUsecase(coreClient, cryptoClient)

	// Initialize handlers
	activitiesHandler := activityhandler.NewActivitiesHandler(activitiesUC)
	escalationsHandler := activityhandler.NewEscalationsHandler(escalationsUC)
	paymentPromisesHandler := activityhandler.NewPaymentPromisesHandler(paymentPromisesUC)
	agentPerformanceHandler := activityhandler.NewAgentPerformanceHandler(agentPerformanceUC)
	notificationsHandler := activityhandler.NewNotificationsHandler(
		notificationsUC,
		cfg.NotificationCursorSecret,
		redisClient,
		cfg.NotificationStreamMaxSeconds,
	)
	dashboardHandler := activityhandler.NewDashboardHandler(dashboardUC)
	authHandler := activityhandler.NewAuthHandler(cfg, oidcClient, sessionStore, flowStateStore)
	healthHandler := activityhandler.NewHealthHandler()
	cryptoSessionMgr, err := fieldcrypto.GetSessionManager()
	if err != nil {
		log.Printf("crypto session manager unavailable: %v", err)
	}
	var tenantAuthority fieldcrypto.TenantAuthority
	if fieldcrypto.SessionModeFromEnv() == "stateless" {
		mgmtClient, mgmtErr := fieldcrypto.ManagementTenantClientFromEnv(coreclient.NewMtlsHTTPClient(5 * time.Second))
		if mgmtErr != nil {
			log.Printf("tenant authority unavailable: %v", mgmtErr)
			tenantAuthority = fieldcrypto.FailClosedTenantAuthority{}
		} else {
			tenantAuthority, err = fieldcrypto.BuildTenantAuthorityFromEnv(mgmtClient)
			if err != nil {
				log.Printf("tenant authority unavailable: %v", err)
				tenantAuthority = fieldcrypto.FailClosedTenantAuthority{}
			} else {
				fieldcrypto.SetTenantAuthority(tenantAuthority)
			}
		}
	}
	cryptoSessionHandler := activityhandler.NewCryptoSessionHandler(cryptoSessionMgr, tenantAuthority)
	cryptoProxyHandler := activityhandler.NewCryptoProxyHandler(cryptoClient)
	auditHandler := activityhandler.NewAuditHandler(coreClient)
	contactsHandler := activityhandler.NewContactsHandler(contactsUC, cryptoClient)

	// Set Gin mode
	gin.SetMode(getEnvOrDefault("GIN_MODE", "release"))

	// Create Gin engine
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestContextMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.RequestTimeoutMiddleware(cfg.RequestTimeoutSeconds))
	r.Use(middleware.RequireJSONContentType())
	r.Use(middleware.RequestSizeLimitMiddleware(cfg.MaxRequestBodyBytes))
	r.Use(middleware.CSRFMiddleware(sessionStore))

	// Middleware pipeline
	r.Use(middleware.TracingMiddleware())
	r.Use(middleware.CognitoContextMiddleware(cognitoValidator))
	r.Use(middleware.RateLimitMiddleware(
		middleware.NewMemoryRateLimitStore(cfg.RateLimitRequests, cfg.RateLimitWindowSec),
		cfg.TrustedProxies,
		cfg.RateLimitSkipPaths,
	))
	r.Use(middleware.AuditMiddleware())
	if cfg.CryptoEnabled {
		cryptoSettings := fieldcrypto.CryptoSettingsFromEnv()
		var fieldCryptoService *fieldcrypto.FieldCryptoService
		switch mgr := cryptoSessionMgr.(type) {
		case *fieldcrypto.CryptoSessionManager:
			fieldCryptoService = fieldcrypto.NewFieldCryptoService(fieldcrypto.NewSessionKeyProvider(mgr))
		default:
			if provider, err := fieldcrypto.EnvKeyProviderFromEnv(); err == nil {
				fieldCryptoService = fieldcrypto.NewFieldCryptoService(provider)
			} else {
				placeholder, _ := fieldcrypto.NewEnvKeyProvider(map[string][]byte{"unused": bytesRepeat(32, 1)}, "unused")
				fieldCryptoService = fieldcrypto.NewFieldCryptoService(placeholder)
			}
		}
		r.Use(middleware.FieldCryptoMiddleware(middleware.FieldCryptoMiddlewareConfig{
			Enabled:         true,
			Service:         fieldCryptoService,
			Policy:          cryptoSettings.Policy(),
			Settings:        cryptoSettings,
			SessionManager:  cryptoSessionMgr,
			TenantAuthority: tenantAuthority,
		}))
	}
	r.Use(middleware.CryptoMiddleware(cfg.CryptoEnabled, middleware.NewCryptoClient(cfg.CryptoBFFBaseURL)))
	r.Use(middleware.CORS(cfg.CORSSOrigins))

	// Register routes
	api.RegisterRoutes(r,
		sessionStore,
		activitiesHandler,
		escalationsHandler,
		paymentPromisesHandler,
		agentPerformanceHandler,
		notificationsHandler,
		dashboardHandler,
		authHandler,
		healthHandler,
		cryptoSessionHandler,
		cryptoProxyHandler,
		auditHandler,
		contactsHandler,
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

func bytesRepeat(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
