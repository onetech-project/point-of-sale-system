package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/tenant-service/api"
)

func main() {
	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/pos_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	registerHandler := api.NewRegisterHandler(db)
	e.POST("/register", registerHandler.Register)

	tenantHandler := api.NewTenantHandler(db)
	e.GET("/tenant", tenantHandler.GetTenant)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("Tenant service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
