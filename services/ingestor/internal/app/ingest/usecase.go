package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"apex/ingestor/internal/app"
	"apex/ingestor/internal/domain"
)

// IngestUseCase orchestrates deduplication, storage, and publishing.
type IngestUseCase struct {
	storage app.Storage
	dedup   app.Deduplicator
	pub     app.Publisher
}

// New creates a new IngestUseCase.
func New(storage app.Storage, dedup app.Deduplicator, pub app.Publisher) *IngestUseCase {
	return &IngestUseCase{storage: storage, dedup: dedup, pub: pub}
}

// Ingest processes a raw invoice file through the full pipeline.
func (uc *IngestUseCase) Ingest(
	ctx context.Context,
	source domain.Source,
	filename string,
	data []byte,
	metadata map[string]string,
) error {
	// 1. Compute SHA-256
	sum := sha256.Sum256(data)
	hashHex := fmt.Sprintf("%x", sum)

	// 2. Deduplication check
	isDup, err := uc.dedup.CheckAndMark(ctx, hashHex)
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if isDup {
		return nil // silent skip
	}

	// 3. Build file key: "{source}/{year}/{month}/{uuid}.{ext}"
	id := uuid.New().String()
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		ext = "bin"
	}
	now := time.Now().UTC()
	fileKey := fmt.Sprintf("%s/%d/%02d/%s.%s", source, now.Year(), now.Month(), id, ext)

	// 4. Detect content type and upload
	contentType := detectContentType(filename, data)
	if err := uc.storage.Upload(ctx, fileKey, contentType, data); err != nil {
		return fmt.Errorf("storage upload: %w", err)
	}

	// 5. Build RawInvoice
	sender := ""
	if metadata != nil {
		if s, ok := metadata["sender"]; ok {
			sender = s
		}
	}

	invoice := domain.RawInvoice{
		ID:         id,
		Source:     source,
		FileKey:    fileKey,
		SHA256:     hashHex,
		Sender:     sender,
		ReceivedAt: now,
		Metadata:   metadata,
	}

	// 6. Publish
	if err := uc.pub.Publish(ctx, invoice); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// detectContentType returns a MIME type based on file extension and magic bytes.
func detectContentType(filename string, data []byte) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	switch ext {
	case "pdf":
		return "application/pdf"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "tiff", "tif":
		return "image/tiff"
	}

	// Magic bytes fallback
	if len(data) >= 4 {
		if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
			return "application/pdf"
		}
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			return "image/png"
		}
		if data[0] == 0xFF && data[1] == 0xD8 {
			return "image/jpeg"
		}
	}

	return "application/octet-stream"
}
