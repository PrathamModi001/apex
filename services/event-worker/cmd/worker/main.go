package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"apex/event-worker/internal/app"
	"apex/event-worker/internal/app/process"
	"apex/event-worker/internal/domain"
	"apex/event-worker/internal/handlers"
	kafkainfra "apex/event-worker/internal/infra/kafka"
	minioinfra "apex/event-worker/internal/infra/minio"
	ocrinfra "apex/event-worker/internal/infra/ocr"
	postgresinfra "apex/event-worker/internal/infra/postgres"
	redisinfra "apex/event-worker/internal/infra/redis"
)

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── 1. Redis → IdempotencyChecker ──────────────────────────────────────
	redisURL := getEnv("REDIS_URL", "redis://redis:6379")
	rdb, err := redisinfra.NewRedisClient(redisURL)
	if err != nil {
		log.Fatalf("init redis: %v", err)
	}
	var idempotency app.IdempotencyChecker = redisinfra.NewIdempotencyChecker(rdb)

	// ── 2. MinIO → FileReader ───────────────────────────────────────────────
	minioEndpoint := getEnv("MINIO_ENDPOINT", "minio:9000")
	minioUser := os.Getenv("MINIO_ROOT_USER")
	minioPwd := os.Getenv("MINIO_ROOT_PASSWORD")
	minioBucket := getEnv("MINIO_BUCKET", "invoices")
	fileReader, err := minioinfra.NewReader(minioEndpoint, minioUser, minioPwd, minioBucket, false)
	if err != nil {
		log.Fatalf("init minio: %v", err)
	}

	// ── 3. Postgres pgx pool → InvoiceWriter + POMatcher ───────────────────
	var invoiceWriter app.InvoiceWriter = &noopInvoiceWriter{}
	var poMatcher app.POMatcher = &noopPOMatcher{}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pool, pgErr := pgxpool.New(ctx, dbURL)
		if pgErr != nil {
			log.Fatalf("init postgres: %v", pgErr)
		}
		defer pool.Close()
		invoiceWriter = postgresinfra.NewInvoiceRepo(pool)
		poMatcher = postgresinfra.NewPOMatcher(pool)
	} else {
		log.Println("WARNING: DATABASE_URL not set — using no-op InvoiceWriter and POMatcher")
	}

	// ── 4. OCR extractor (Groq HTTP client with GROQ_API_KEY) ──────────────
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	var ocrExtractor app.OCRExtractor = ocrinfra.New(groqAPIKey)

	// ── 5. Kafka writer → EventPublisher ───────────────────────────────────
	brokersRaw := getEnv("KAFKA_BROKERS", "redpanda:29092")
	brokers := strings.Split(brokersRaw, ",")
	topicProcessed := getEnv("KAFKA_TOPIC_PROCESSED", "invoice.processed")
	publisher := kafkainfra.NewPublisher(brokers, topicProcessed)
	defer func() {
		if closeErr := publisher.Close(); closeErr != nil {
			log.Printf("kafka publisher close: %v", closeErr)
		}
	}()

	// ── 6. Wire ProcessUseCase ──────────────────────────────────────────────
	uc := process.New(fileReader, ocrExtractor, poMatcher, idempotency, invoiceWriter, publisher)

	// ── 7. Kafka reader + consumer goroutine ────────────────────────────────
	topicRaw := getEnv("KAFKA_TOPIC_RAW", "invoice.raw")
	consumer := kafkainfra.NewConsumer(brokers, "event-worker", topicRaw, uc)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("starting kafka consumer on topic %s", topicRaw)
		consumer.Run(ctx)
		log.Println("kafka consumer stopped")
	}()

	// ── 8. Echo HTTP server goroutine ───────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.GET("/health", handlers.HealthHandler())

	port := getEnv("PORT", "8082")

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("starting HTTP server on :%s", port)
		if httpErr := e.Start(":" + port); httpErr != nil {
			log.Printf("HTTP server stopped: %v", httpErr)
		}
	}()

	// ── 9. Wait for SIGINT/SIGTERM → cancel → wait for goroutines ──────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received")

	cancel() // stops Kafka consumer loop

	shutCtx := context.Background()
	if shutErr := e.Shutdown(shutCtx); shutErr != nil {
		log.Printf("HTTP server shutdown error: %v", shutErr)
	}

	wg.Wait()
	log.Println("event-worker stopped")
}

// ── no-op adapters for local dev without a real DB ─────────────────────────

type noopInvoiceWriter struct{}

func (n *noopInvoiceWriter) Create(_ context.Context, inv domain.ProcessedInvoice) error {
	log.Printf("NOOP InvoiceWriter: would write invoice %s", inv.ID)
	return nil
}

type noopPOMatcher struct{}

func (n *noopPOMatcher) Match(_ context.Context, vendorName string, amount float64) (domain.POMatch, error) {
	log.Printf("NOOP POMatcher: vendor=%q amount=%f", vendorName, amount)
	return domain.POMatch{Matched: false, Confidence: 0}, nil
}
