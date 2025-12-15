package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/auth-service/api"
	"github.com/pos/auth-service/src/config"
	"github.com/pos/auth-service/src/queue"
	"github.com/pos/auth-service/src/services"
	"github.com/pos/auth-service/src/utils"
)

func main() {
	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = true

	// Set validator
	e.Validator = utils.NewValidator()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Database connection
	db, err := sql.Open("postgres", config.DB_URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.REDIS_HOST,
		Password: config.REDIS_PASSWORD,
		DB:       config.REDIS_DB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize services
	sessionManager := services.NewSessionManager(redisClient, config.SESSION_TTL_MINUTES)

	jwtService := services.NewJWTService(config.JWT_SECRET, config.JWT_EXPIRATION_MINUTES)

	rateLimiter := services.NewRateLimiter(redisClient, config.RATE_LIMIT_LOGIN_MAX, config.RATE_LIMIT_LOGIN_WINDOW)

	// Initialize Kafka producer and event publisher
	eventPublisher := queue.NewEventPublisher(config.KAFKA_BROKERS)
	defer eventPublisher.Close()

	authService := services.NewAuthService(db, sessionManager, jwtService, rateLimiter, eventPublisher)

	// Health checks
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Auth endpoints
	loginHandler := api.NewLoginHandler(authService)
	e.POST("/login", loginHandler.Login)

	sessionHandler := api.NewSessionHandler(authService, jwtService)
	e.GET("/session", sessionHandler.GetSession)

	logoutHandler := api.NewLogoutHandler(authService, jwtService)
	e.POST("/logout", logoutHandler.Logout)

	// Password reset endpoints
	passwordResetService := services.NewPasswordResetService(db, eventPublisher)
	passwordResetHandler := api.NewPasswordResetHandler(passwordResetService)
	e.POST("/password-reset/request", passwordResetHandler.RequestReset)
	e.POST("/password-reset/reset", passwordResetHandler.ResetPassword)

	// Start server
	log.Printf("Auth service starting on port %s", config.PORT)
	e.Logger.Fatal(e.Start(":" + config.PORT))
}
