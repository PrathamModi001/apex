package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/segmentio/kafka-go"

	"apex/ingestor/internal/domain"
)

// KafkaPublisher publishes RawInvoice messages to a Kafka topic.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// New creates a KafkaPublisher writing to the given topic via the provided brokers.
// brokers is a comma-separated list, e.g. "redpanda:29092".
func New(brokers, topic string) *KafkaPublisher {
	addrs := strings.Split(brokers, ",")
	w := &kafka.Writer{
		Addr:     kafka.TCP(addrs...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaPublisher{writer: w}
}

// Publish serialises the RawInvoice as JSON and writes it to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, invoice domain.RawInvoice) error {
	payload, err := json.Marshal(invoice)
	if err != nil {
		return fmt.Errorf("kafka marshal invoice: %w", err)
	}
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(invoice.ID),
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}
	return nil
}

// Close closes the underlying Kafka writer.
func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
