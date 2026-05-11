package main

import (
	"database/sql"
	"log"
	"strings"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"

	"github.com/pos/billing-service/api"
	subcache "github.com/pos/billing-service/src/cache"
	"github.com/pos/billing-service/src/jobs"
	"github.com/pos/billing-service/src/queue"
	"github.com/pos/billing-service/src/repository"
	"github.com/pos/billing-service/src/services"
	. "github.com/pos/billing-service/src/utils"
)

func main() {
	e := echo.New()
	e.Use(emw.Recover())
	e.Use(emw.Logger())

	dbURL := GetEnv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	kafkaBrokers := strings.Split(GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := GetEnv("KAFKA_TOPIC")
	publisher := queue.NewEventPublisher(kafkaBrokers, kafkaTopic)
	defer publisher.Close()

	repo := repository.NewBillingRepository(db)
	paymentSvc := services.NewPaymentService()
	subSvc := services.NewSubscriptionService(db, repo, publisher, paymentSvc)
	subscriptionCache := subcache.NewSubscriptionCacheFromEnv()
	if subscriptionCache != nil {
		defer subscriptionCache.Close()
		subSvc.SetSubscriptionCacheInvalidator(subscriptionCache)
	}
	handler := api.NewBillingHandler(subSvc)

	// Health
	e.GET("/health", handler.Health)
	e.GET("/ready", handler.Ready)

	// Internal (no auth - only accessible from within the cluster)
	e.GET("/internal/subscription/:tenant_id", handler.GetInternalSubscriptionStatus)

	// Protected routes (tenant_id injected as X-Tenant-ID header by the API gateway)
	billing := e.Group("/api/v1/billing")
	billing.GET("/subscription", handler.GetMySubscription)
	billing.PUT("/subscription/cycle", handler.UpdateBillingCycle)
	billing.POST("/subscription/upgrade", handler.UpgradeSubscription)
	billing.GET("/invoices", handler.ListInvoices)
	billing.GET("/invoices/:id", handler.GetInvoice)
	billing.POST("/invoices/:id/pay", handler.InitiatePayment)

	// Webhook (no auth, signature verified inside handler)
	e.POST("/webhook/billing", handler.HandleMidtransWebhook)

	// Start background jobs
	jobRunner := jobs.NewJobRunner(db, repo, publisher)
	if subscriptionCache != nil {
		jobRunner.SetSubscriptionCacheInvalidator(subscriptionCache)
	}
	go jobRunner.StartAll()

	port := GetEnv("PORT")
	log.Printf("Billing service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
