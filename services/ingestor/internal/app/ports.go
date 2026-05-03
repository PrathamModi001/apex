package app

import (
	"context"

	"apex/ingestor/internal/domain"
)

// Storage persists raw invoice files.
type Storage interface {
	Upload(ctx context.Context, key, contentType string, data []byte) error
}

// Deduplicator detects and marks previously-seen SHA-256 digests.
type Deduplicator interface {
	// CheckAndMark returns (isDuplicate=true, nil) if the SHA-256 was already seen.
	// Returns (false, nil) and marks the key as seen when the hash is new.
	CheckAndMark(ctx context.Context, sha256 string) (bool, error)
}

// Publisher sends a RawInvoice to the downstream message bus.
type Publisher interface {
	Publish(ctx context.Context, invoice domain.RawInvoice) error
	Close() error
}
