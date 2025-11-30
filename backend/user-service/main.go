package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/user-service/api"
)

func main() {
	e := echo.New()

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

	// Health checks
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Invitation endpoints
	invitationHandler := api.NewInvitationHandler(db)
	e.POST("/invitations", invitationHandler.CreateInvitation)
	e.GET("/invitations", invitationHandler.ListInvitations)
	e.POST("/invitations/:token/accept", invitationHandler.AcceptInvitation)

	// Start server
	port := getEnv("PORT", "8083")
	log.Printf("User service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
