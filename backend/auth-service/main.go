package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/auth-service/api"
	"github.com/pos/auth-service/src/services"
)

func main() {
	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/pos_db?sslmode=disable")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis connection
	redisHost := getEnv("REDIS_HOST", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
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
	sessionTTL := getEnvInt("SESSION_TTL_MINUTES", 15)
	sessionManager := services.NewSessionManager(redisClient, sessionTTL)

	jwtSecret := getEnv("JWT_SECRET", "default-secret-change-in-production")
	jwtExpiration := getEnvInt("JWT_EXPIRATION_MINUTES", 15)
	jwtService := services.NewJWTService(jwtSecret, jwtExpiration)

	rateLimitMax := getEnvInt("RATE_LIMIT_LOGIN_MAX", 5)
	rateLimitWindow := getEnvInt("RATE_LIMIT_LOGIN_WINDOW", 900) // 15 minutes in seconds
	rateLimiter := services.NewRateLimiter(redisClient, rateLimitMax, rateLimitWindow)

	authService := services.NewAuthService(db, sessionManager, jwtService, rateLimiter)

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

	// Start server
	port := getEnv("PORT", "8082")
	log.Printf("Auth service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}

	return intValue
}
