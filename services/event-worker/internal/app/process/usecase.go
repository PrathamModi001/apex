package process

import (
	"context"
	"fmt"
	"path"
	"time"

	"apex/event-worker/internal/app"
	"apex/event-worker/internal/domain"
)

// ProcessUseCase orchestrates the full invoice processing pipeline.
type ProcessUseCase struct {
	fileReader  app.FileReader
	ocr         app.OCRExtractor
	poMatcher   app.POMatcher
	idempotency app.IdempotencyChecker
	writer      app.InvoiceWriter
	publisher   app.EventPublisher
}

// New creates a new ProcessUseCase with all required dependencies.
func New(
	fileReader app.FileReader,
	ocr app.OCRExtractor,
	poMatcher app.POMatcher,
	idempotency app.IdempotencyChecker,
	writer app.InvoiceWriter,
	publisher app.EventPublisher,
) *ProcessUseCase {
	return &ProcessUseCase{
		fileReader:  fileReader,
		ocr:         ocr,
		poMatcher:   poMatcher,
		idempotency: idempotency,
		writer:      writer,
		publisher:   publisher,
	}
}

// Process runs the full invoice processing pipeline for one raw invoice.
func (uc *ProcessUseCase) Process(ctx context.Context, raw domain.RawInvoice) error {
	// 1. Idempotency check: key = raw.SHA256 + ":" + raw.ID
	//    if already seen → return nil (silent skip)
	idemKey := raw.SHA256 + ":" + raw.ID
	alreadySeen, err := uc.idempotency.CheckAndMark(ctx, idemKey)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if alreadySeen {
		return nil
	}

	// 2. Download file from MinIO using raw.FileKey
	data, _, err := uc.fileReader.Download(ctx, raw.FileKey)
	if err != nil {
		return fmt.Errorf("download file %q: %w", raw.FileKey, err)
	}

	// 3. OCR extract fields (filename from last segment of FileKey)
	filename := path.Base(raw.FileKey)
	fields, err := uc.ocr.Extract(ctx, data, filename)
	if err != nil {
		return fmt.Errorf("ocr extract: %w", err)
	}

	// 4. PO match using extracted vendor name + amount
	poMatch, err := uc.poMatcher.Match(ctx, fields.VendorName, fields.Amount)
	if err != nil {
		return fmt.Errorf("po match: %w", err)
	}

	// 5. Build ProcessedInvoice
	processed := domain.ProcessedInvoice{
		ID:              raw.ID,
		Source:          raw.Source,
		FileKey:         raw.FileKey,
		SHA256:          raw.SHA256,
		Sender:          raw.Sender,
		ReceivedAt:      raw.ReceivedAt,
		ExtractedFields: fields,
		POMatch:         poMatch,
		ProcessedAt:     time.Now().UTC(),
	}

	// 6. Write to Postgres (invoices table)
	if err := uc.writer.Create(ctx, processed); err != nil {
		return fmt.Errorf("write invoice: %w", err)
	}

	// 7. Publish to invoice.processed
	if err := uc.publisher.Publish(ctx, processed); err != nil {
		return fmt.Errorf("publish invoice: %w", err)
	}

	return nil
}
