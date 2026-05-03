package kafka

import (
	"context"
	"encoding/json"
	"log"

	kafkago "github.com/segmentio/kafka-go"

	"apex/event-worker/internal/app/process"
	"apex/event-worker/internal/domain"
)

// Consumer reads messages from invoice.raw and drives ProcessUseCase.
type Consumer struct {
	reader *kafkago.Reader
	uc     *process.ProcessUseCase
}

// NewConsumer creates a Kafka consumer bound to the given brokers/group/topic.
func NewConsumer(brokers []string, groupID, topic string, uc *process.ProcessUseCase) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, uc: uc}
}

// Run starts the consumer loop. Blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	defer func() {
		if err := c.reader.Close(); err != nil {
			log.Printf("kafka consumer close error: %v", err)
		}
	}()

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			// Context cancelled or broker error — stop the loop
			if ctx.Err() != nil {
				log.Printf("kafka consumer stopping: %v", ctx.Err())
				return
			}
			log.Printf("kafka read error: %v", err)
			continue
		}

		var raw domain.RawInvoice
		if jsonErr := json.Unmarshal(m.Value, &raw); jsonErr != nil {
			log.Printf("kafka message unmarshal error (offset %d): %v", m.Offset, jsonErr)
			continue
		}

		if processErr := c.uc.Process(ctx, raw); processErr != nil {
			log.Printf("process error for invoice %s: %v", raw.ID, processErr)
			// continue consuming (don't re-queue, just log)
		}
	}
}
