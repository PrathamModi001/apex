package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"

	"apex/api-gateway/internal/app"
)

// Consumer reads from a Kafka topic and routes events to the Hub and Notifier.
type Consumer struct {
	reader   *kafka.Reader
	hub      app.EventBus
	notifier app.TelegramNotifier
}

// NewConsumer creates a Consumer for the given Kafka brokers and topic.
// brokers is a comma-separated list (e.g. "kafka:9092,kafka2:9092").
func NewConsumer(brokers, topic string, hub app.EventBus, notifier app.TelegramNotifier) *Consumer {
	brokerList := strings.Split(brokers, ",")
	for i, b := range brokerList {
		brokerList[i] = strings.TrimSpace(b)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokerList,
		Topic:    topic,
		GroupID:  "api-gateway",
		MinBytes: 1,
		MaxBytes: 10e6, // 10 MB
	})

	return &Consumer{
		reader:   reader,
		hub:      hub,
		notifier: notifier,
	}
}

// Start reads messages in a loop until ctx is cancelled. Call in a goroutine.
func (c *Consumer) Start(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled — clean shutdown.
				return
			}
			log.Printf("kafka consumer: read error: %v", err)
			continue
		}

		var event app.DecisionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("kafka consumer: unmarshal error: %v", err)
			continue
		}

		c.hub.Broadcast(event)

		if event.RiskScore > 60 {
			if err := c.notifier.SendApprovalRequest(
				"", // use notifier's configured chatID
				event.InvoiceID,
				event.VendorName,
				0, // amount not carried in DecisionEvent
				event.RiskScore,
				"High-risk invoice requires manual review",
			); err != nil {
				log.Printf("kafka consumer: telegram notify: %v", err)
			}
		}
	}
}

// Close shuts down the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
