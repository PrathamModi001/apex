package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/handlers"
	gmailinfra "apex/ingestor/internal/infra/gmail"
	kafkainfra "apex/ingestor/internal/infra/kafka"
	minioinfra "apex/ingestor/internal/infra/minio"
	redisinfra "apex/ingestor/internal/infra/redis"
	telegraminfra "apex/ingestor/internal/infra/telegram"

	goredis "github.com/redis/go-redis/v9"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// -------------------------------------------------------------------------
	// Environment
	// -------------------------------------------------------------------------
	kafkaBrokers := getenv("KAFKA_BROKERS", "redpanda:29092")
	kafkaTopic := getenv("KAFKA_TOPIC_RAW", "invoice.raw")
	minioEndpoint := getenv("MINIO_ENDPOINT", "minio:9000")
	minioUser := getenv("MINIO_ROOT_USER", "minioadmin")
	minioPass := getenv("MINIO_ROOT_PASSWORD", "minioadmin")
	minioBucket := getenv("MINIO_BUCKET", "invoices")
	redisURL := getenv("REDIS_URL", "redis://redis:6379")
	databaseURL := os.Getenv("DATABASE_URL")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURI := getenv("GOOGLE_REDIRECT_URI", "http://localhost:8081/auth/google/callback")
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramWebhookSecret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	port := getenv("PORT", "8081")

	// -------------------------------------------------------------------------
	// 1. Redis → Deduplicator
	// -------------------------------------------------------------------------
	redisOpts, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("parse REDIS_URL: %v", err)
	}
	rdb := goredis.NewClient(redisOpts)
	dedup := redisinfra.NewDeduplicator(rdb)

	// -------------------------------------------------------------------------
	// 2. MinIO → Storage
	// -------------------------------------------------------------------------
	stor, err := minioinfra.New(minioEndpoint, minioUser, minioPass, minioBucket, false)
	if err != nil {
		log.Fatalf("init minio: %v", err)
	}

	// -------------------------------------------------------------------------
	// 3. Kafka → Publisher
	// -------------------------------------------------------------------------
	brokerList := strings.Join(strings.Split(kafkaBrokers, ","), ",")
	pub := kafkainfra.New(brokerList, kafkaTopic)

	// -------------------------------------------------------------------------
	// 4. pgx pool for google_tokens
	// -------------------------------------------------------------------------
	var pgPool *pgxpool.Pool
	if databaseURL != "" {
		pgPool, err = pgxpool.New(ctx, databaseURL)
		if err != nil {
			log.Fatalf("init pgx pool: %v", err)
		}
		defer pgPool.Close()
	} else {
		log.Println("[main] DATABASE_URL not set — Gmail OAuth disabled")
	}

	// -------------------------------------------------------------------------
	// 5. IngestUseCase
	// -------------------------------------------------------------------------
	ingestUC := ingest.New(stor, dedup, pub)

	// -------------------------------------------------------------------------
	// 6. Gmail poller (optional — needs OAuth creds + DB)
	// -------------------------------------------------------------------------
	if googleClientID != "" && googleClientSecret != "" && pgPool != nil {
		oauthCfg := gmailinfra.OAuthConfig(googleClientID, googleClientSecret, googleRedirectURI)
		poller := gmailinfra.NewPoller(ingestUC, pgPool, oauthCfg)
		poller.Start(ctx)
	} else {
		log.Println("[main] Gmail OAuth creds or DB missing — poller disabled")
	}

	// -------------------------------------------------------------------------
	// 7. Telegram webhook handler (optional — needs bot token)
	// -------------------------------------------------------------------------
	var telegramWH *telegraminfra.WebhookHandler
	if telegramBotToken != "" {
		telegramWH = telegraminfra.NewWebhookHandler(ingestUC, telegramBotToken, telegramWebhookSecret)
	} else {
		log.Println("[main] TELEGRAM_BOT_TOKEN not set — /telegram/webhook disabled")
	}

	// -------------------------------------------------------------------------
	// 8. Google OAuth config for auth routes
	// -------------------------------------------------------------------------
	var oauthCfgForRoutes interface{ AuthCodeURL(string, ...interface{}) string }
	_ = oauthCfgForRoutes // used below in routes if available

	// -------------------------------------------------------------------------
	// 9. Echo server
	// -------------------------------------------------------------------------
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", handlers.HealthHandler())

	if googleClientID != "" && googleClientSecret != "" && pgPool != nil {
		oauthCfg := gmailinfra.OAuthConfig(googleClientID, googleClientSecret, googleRedirectURI)
		e.GET("/auth/google", handlers.GoogleAuthHandler(oauthCfg))
		e.GET("/auth/google/callback", handlers.GoogleCallbackHandler(oauthCfg, pgPool))
	}

	if telegramWH != nil {
		e.POST("/telegram/webhook", handlers.TelegramWebhookHandler(telegramWH))
	}

	e.POST("/ingest/test", handlers.TestIngestHandler(ingestUC))

	// -------------------------------------------------------------------------
	// 10. Graceful shutdown
	// -------------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[main] shutting down...")
		cancel()                       // stop poller
		if err := pub.Close(); err != nil { // close kafka writer
			log.Printf("[main] kafka close: %v", err)
		}
		if err := rdb.Close(); err != nil {
			log.Printf("[main] redis close: %v", err)
		}
	}()

	log.Fatal(e.Start(":" + port))
}
