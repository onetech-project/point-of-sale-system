package main

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/auth-service/api"
	"github.com/pos/auth-service/middleware"
	"github.com/pos/auth-service/src/observability"
	"github.com/pos/auth-service/src/queue"
	"github.com/pos/auth-service/src/repository"
	"github.com/pos/auth-service/src/services"
	"github.com/pos/auth-service/src/utils"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func main() {
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(nil)

	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = utils.GetEnvBool("DEBUG")

	// Set validator
	e.Validator = utils.NewValidator()

	e.Use(emw.Recover())

	// OTEL
	e.Use(otelecho.Middleware(utils.GetEnv("SERVICE_NAME")))

	// Trace → Log bridge
	e.Use(middleware.TraceLogger)

	// Logging with PII masking (T061)
	e.Use(middleware.LoggingMiddleware)

	middleware.MetricsMiddleware(e)

	// Database connection
	dbURL := utils.GetEnv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection
	redisHost := utils.GetEnv("REDIS_HOST")
	redisPassword := utils.GetEnv("REDIS_PASSWORD")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       0,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize services
	sessionTTL := utils.GetEnvInt("SESSION_TTL_MINUTES")
	sessionManager := services.NewSessionManager(redisClient, sessionTTL)

	jwtSecret := utils.GetEnv("JWT_SECRET")
	jwtExpiration := utils.GetEnvInt("JWT_EXPIRATION_MINUTES")
	jwtService := services.NewJWTService(jwtSecret, jwtExpiration)

	rateLimitMax := utils.GetEnvInt("RATE_LIMIT_LOGIN_MAX")
	rateLimitWindow := utils.GetEnvInt("RATE_LIMIT_LOGIN_WINDOW")
	rateLimiter := services.NewRateLimiter(redisClient, rateLimitMax, rateLimitWindow)

	// Initialize Kafka producer and event publisher
	kafkaBrokers := strings.Split(utils.GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := utils.GetEnv("KAFKA_TOPIC")
	eventPublisher := queue.NewEventPublisher(kafkaBrokers, kafkaTopic)
	defer eventPublisher.Close()

	// Initialize AuditPublisher for audit trail (T103, T104)
	auditTopic := utils.GetEnv("KAFKA_AUDIT_TOPIC")
	serviceName := utils.GetEnv("SERVICE_NAME")
	auditPublisher, err := utils.NewAuditPublisher(serviceName, kafkaBrokers, auditTopic)
	if err != nil {
		log.Fatalf("Failed to initialize AuditPublisher: %v", err)
	}
	defer auditPublisher.Close()

	authService, err := services.NewAuthService(db, sessionManager, jwtService, rateLimiter, eventPublisher, auditPublisher)
	if err != nil {
		log.Fatalf("Failed to initialize AuthService: %v", err)
	}

	// Initialize VaultClient for password reset service
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		log.Fatalf("Failed to initialize VaultClient for password reset: %v", err)
	}

	// Health checks
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Auth endpoints
	loginHandler := api.NewLoginHandler(authService)
	e.POST("/login", loginHandler.Login)

	sessionHandler := api.NewSessionHandler(authService, jwtService)
	e.GET("/session", sessionHandler.GetSession)
	e.POST("/refresh", sessionHandler.RefreshSession)

	logoutHandler := api.NewLogoutHandler(authService, jwtService)
	e.POST("/logout", logoutHandler.Logout)

	// Account verification endpoints
	accountVerificationHandler := api.NewAccountVerificationHandler(authService)
	e.POST("/verify-account", accountVerificationHandler.VerifyAccount)

	// Password reset endpoints
	passwordResetRepo, err := repository.NewPasswordResetRepositoryWithVault(db)
	if err != nil {
		log.Fatalf("Failed to initialize PasswordResetRepository: %v", err)
	}
	passwordResetService := services.NewPasswordResetService(passwordResetRepo, db, eventPublisher, vaultClient)
	passwordResetHandler := api.NewPasswordResetHandler(passwordResetService)
	e.POST("/password-reset/request", passwordResetHandler.RequestReset)
	e.POST("/password-reset/reset", passwordResetHandler.ResetPassword)

	// Start server
	port := utils.GetEnv("PORT")
	log.Printf("Auth service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
