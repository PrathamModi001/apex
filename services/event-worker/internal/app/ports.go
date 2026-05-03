package app

import (
	"context"

	"apex/event-worker/internal/domain"
)

// FileReader downloads a file from object storage.
type FileReader interface {
	Download(ctx context.Context, key string) ([]byte, string, error) // data, contentType, err
}

// OCRExtractor extracts structured fields from raw document bytes.
type OCRExtractor interface {
	Extract(ctx context.Context, data []byte, filename string) (domain.ExtractedFields, error)
}

// POMatcher matches an invoice to a purchase order.
type POMatcher interface {
	Match(ctx context.Context, vendorName string, amount float64) (domain.POMatch, error)
}

// IdempotencyChecker prevents re-processing of already-seen invoices.
// CheckAndMark returns (alreadySeen, error). Marks key as seen if new.
type IdempotencyChecker interface {
	CheckAndMark(ctx context.Context, key string) (bool, error)
}

// InvoiceWriter persists a processed invoice to the database.
type InvoiceWriter interface {
	Create(ctx context.Context, inv domain.ProcessedInvoice) error
}

// EventPublisher publishes processed invoices to downstream topics.
type EventPublisher interface {
	Publish(ctx context.Context, inv domain.ProcessedInvoice) error
	Close() error
}
