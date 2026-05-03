package kafka_test

import (
	"context"
	"testing"
	"time"

	"apex/ingestor/internal/domain"
	kafkainfra "apex/ingestor/internal/infra/kafka"
)

// TestKafkaPublisher_Close verifies that Close does not panic when called on
// a publisher that was never connected (unit smoke test — no real Kafka needed).
func TestKafkaPublisher_Close(t *testing.T) {
	p := kafkainfra.New("localhost:9092", "test-topic")
	if err := p.Close(); err != nil {
		// Closing a writer that was never connected may return an error on some
		// versions; we just want to ensure it does not panic.
		t.Logf("Close returned (non-fatal): %v", err)
	}
}

// TestKafkaPublisher_PublishTimeout verifies that Publish respects context
// cancellation rather than blocking forever.
func TestKafkaPublisher_PublishTimeout(t *testing.T) {
	p := kafkainfra.New("localhost:19999", "test-topic") // unreachable broker

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	inv := domain.RawInvoice{
		ID:         "test-id",
		Source:     domain.SourceTest,
		FileKey:    "test/2024/01/test-id.pdf",
		SHA256:     "abc123",
		ReceivedAt: time.Now(),
	}

	err := p.Publish(ctx, inv)
	if err == nil {
		t.Fatal("expected error when connecting to unreachable broker, got nil")
	}
	_ = p.Close()
}
