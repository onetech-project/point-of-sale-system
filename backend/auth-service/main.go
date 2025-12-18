package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/auth-service/api"
	"github.com/pos/auth-service/src/queue"
	"github.com/pos/auth-service/src/repository"
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
	dbURL := getEnv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection
	redisHost := getEnv("REDIS_HOST")
	redisPassword := getEnv("REDIS_PASSWORD")
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
	sessionTTL := getEnvInt("SESSION_TTL_MINUTES")
	sessionManager := services.NewSessionManager(redisClient, sessionTTL)

	jwtSecret := getEnv("JWT_SECRET")
	jwtExpiration := getEnvInt("JWT_EXPIRATION_MINUTES")
	jwtService := services.NewJWTService(jwtSecret, jwtExpiration)

	rateLimitMax := getEnvInt("RATE_LIMIT_LOGIN_MAX")
	rateLimitWindow := getEnvInt("RATE_LIMIT_LOGIN_WINDOW")
	rateLimiter := services.NewRateLimiter(redisClient, rateLimitMax, rateLimitWindow)

	// Initialize Kafka producer and event publisher
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC")
	eventPublisher := queue.NewEventPublisher(kafkaBrokers, kafkaTopic)
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

	// Account verification endpoints
	accountVerificationHandler := api.NewAccountVerificationHandler(authService)
	e.POST("/verify-account", accountVerificationHandler.VerifyAccount)

	// Password reset endpoints
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	passwordResetService := services.NewPasswordResetService(passwordResetRepo, db, eventPublisher)
	passwordResetHandler := api.NewPasswordResetHandler(passwordResetService)
	e.POST("/password-reset/request", passwordResetHandler.RequestReset)
	e.POST("/password-reset/reset", passwordResetHandler.ResetPassword)

	// Start server
	port := getEnv("PORT")
	log.Printf("Auth service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// throw error: required environment variable not set
		panic(key + " environment variable is not set")
	}
	return value
}

func getEnvInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		// throw error: required environment variable not set
		panic(key + " environment variable is not set")
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		// throw error: invalid integer value
		panic("Invalid integer value for " + key)
	}

	return intValue
}
