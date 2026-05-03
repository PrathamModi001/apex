package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"apex/event-worker/internal/domain"
)

// InvoiceRepo writes processed invoices to the Postgres invoices table.
type InvoiceRepo struct {
	pool *pgxpool.Pool
}

// NewInvoiceRepo creates an InvoiceRepo using the provided connection pool.
func NewInvoiceRepo(pool *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{pool: pool}
}

// Create inserts a processed invoice record into the invoices table.
func (r *InvoiceRepo) Create(ctx context.Context, inv domain.ProcessedInvoice) error {
	const q = `
		INSERT INTO invoices (
			id, source, file_key, sha256, sender, received_at,
			invoice_no, amount, currency, due_date, vendor_name,
			po_id, po_confidence, po_matched,
			processed_at, status
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, 'PROCESSING'
		)
		ON CONFLICT (id) DO NOTHING`

	_, err := r.pool.Exec(ctx, q,
		inv.ID,
		inv.Source,
		inv.FileKey,
		inv.SHA256,
		inv.Sender,
		inv.ReceivedAt,
		inv.ExtractedFields.InvoiceNo,
		inv.ExtractedFields.Amount,
		inv.ExtractedFields.Currency,
		inv.ExtractedFields.DueDate,
		inv.ExtractedFields.VendorName,
		inv.POMatch.POID,
		inv.POMatch.Confidence,
		inv.POMatch.Matched,
		inv.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("insert invoice %q: %w", inv.ID, err)
	}
	return nil
}
