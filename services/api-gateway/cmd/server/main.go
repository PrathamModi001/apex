package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	goredis "github.com/redis/go-redis/v9"

	"apex/api-gateway/internal/app/auth"
	"apex/api-gateway/internal/app/invoice"
	"apex/api-gateway/internal/domain"
	"apex/api-gateway/internal/handlers"
	kafkainfra "apex/api-gateway/internal/infra/kafka"
	postgresinfra "apex/api-gateway/internal/infra/postgres"
	redisinfra "apex/api-gateway/internal/infra/redis"
	"apex/api-gateway/internal/infra/telegram"
	"apex/api-gateway/internal/infra/ws"
	apimw "apex/api-gateway/internal/middleware"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	ctx := context.Background()

	// --- Infrastructure ---

	// PostgreSQL
	databaseURL := getEnv("DATABASE_URL", "postgres://apex:apex@localhost:5432/apex?sslmode=disable")
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("pgx pool: %v", err)
	}
	defer pool.Close()

	userRepo := postgresinfra.NewUserRepo(pool)
	invoiceRepo := postgresinfra.NewInvoiceRepo(pool)
	auditRepo := postgresinfra.NewAuditRepo(pool)

	// Redis
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	redisOpts, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis parse URL: %v", err)
	}
	redisClient := goredis.NewClient(redisOpts)
	_ = redisinfra.NewRateLimiter(redisClient) // available for future rate-limiting middleware

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// Telegram Notifier
	telegramToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	telegramChatID := getEnv("TELEGRAM_ADMIN_CHAT_ID", "")
	notifier := telegram.NewNotifier(telegramToken, telegramChatID)

	// Kafka Consumer
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	consumer := kafkainfra.NewConsumer(kafkaBrokers, "invoice.decision", hub, notifier)
	go consumer.Start(ctx)
	defer consumer.Close() //nolint:errcheck

	// --- Use Cases ---
	authUC := auth.New(userRepo)
	invoiceUC := invoice.New(invoiceRepo)

	// --- HTTP Layer ---
	e := echo.New()
	e.HideBanner = true
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(apimw.JWTMiddleware(authUC))

	authH := handlers.NewAuthHandler(authUC)
	invoiceH := handlers.NewInvoiceHandler(invoiceUC, hub)

	// Public routes
	e.GET("/health", handlers.HealthHandler())
	e.GET("/auth/google", authH.GoogleLogin)
	e.GET("/auth/google/callback", authH.GoogleCallback)
	e.GET("/ws", invoiceH.WSHandler)

	// Invoice routes — any authenticated role
	anyRole := apimw.RequireRole(domain.RoleAdmin, domain.RoleReviewer, domain.RoleViewer)
	e.GET("/invoices", invoiceH.ListInvoices, anyRole)
	e.GET("/invoices/:id", invoiceH.GetInvoice, anyRole)
	e.GET("/invoices/:id/decision", invoiceH.GetDecision, anyRole)

	// Invoice mutation routes — reviewer or admin
	reviewerOrAdmin := apimw.RequireRole(domain.RoleAdmin, domain.RoleReviewer)
	e.POST("/invoices/:id/approve", invoiceH.ApproveInvoice, reviewerOrAdmin)
	e.POST("/invoices/:id/reject", invoiceH.RejectInvoice, reviewerOrAdmin)

	// Audit chain — any authenticated role
	e.GET("/audit/:id", handlers.GetAuditChainHandler(auditRepo), anyRole)
	e.POST("/invoices/:id/verify-chain", handlers.VerifyChainHandler(auditRepo), anyRole)

	// User listing — any authenticated role
	e.GET("/users", handlers.ListUsersHandler(userRepo), anyRole)

	// Vendor routes — any authenticated role
	e.GET("/vendors", handlers.ListVendorsHandler(pool), anyRole)
	e.GET("/vendors/:id", handlers.GetVendorHandler(pool), anyRole)

	// Admin routes — admin only
	adminOnly := apimw.RequireRole(domain.RoleAdmin)
	e.POST("/admin/users/:id/role", handlers.UpdateUserRoleHandler(userRepo), adminOnly)

	// Policy routes — reads: any role; writes: reviewer+; delete: admin
	e.GET("/policies", handlers.ListPoliciesHandler(pool), anyRole)
	agentServiceURL := getEnv("AGENT_SERVICE_URL", "http://agent-service:8000")
	e.POST("/policies", handlers.CreatePolicyHandler(agentServiceURL), reviewerOrAdmin)
	e.PATCH("/policies/:id", handlers.TogglePolicyHandler(pool), reviewerOrAdmin)
	e.DELETE("/policies/:id", handlers.DeletePolicyHandler(pool), adminOnly)

	port := getEnv("PORT", "8080")
	log.Fatal(e.Start(":" + port))
}
