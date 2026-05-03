package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"

	"apex/event-worker/internal/domain"
)

// Publisher writes processed invoices to the invoice.processed topic.
type Publisher struct {
	writer *kafkago.Writer
}

// NewPublisher creates a Kafka publisher targeting the given brokers and topic.
func NewPublisher(brokers []string, topic string) *Publisher {
	w := &kafkago.Writer{
		Addr:     kafkago.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafkago.LeastBytes{},
	}
	return &Publisher{writer: w}
}

// Publish serialises the ProcessedInvoice and writes it to Kafka.
func (p *Publisher) Publish(ctx context.Context, inv domain.ProcessedInvoice) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal processed invoice: %w", err)
	}
	msg := kafkago.Message{
		Key:   []byte(inv.ID),
		Value: data,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka write message: %w", err)
	}
	return nil
}

// Close shuts down the underlying Kafka writer.
func (p *Publisher) Close() error {
	return p.writer.Close()
}
