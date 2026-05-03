package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"apex/api-gateway/internal/app"
)

// AuditRepo reads the append-only audit_log table.
type AuditRepo struct {
	pool *pgxpool.Pool
}

// NewAuditRepo creates an AuditRepo.
func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

// GetChain returns the full audit chain for an invoice in insertion order.
func (r *AuditRepo) GetChain(ctx context.Context, invoiceID string) ([]app.AuditEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, invoice_id, event_type, actor, payload, prev_hash, chain_hash, created_at
		 FROM audit_log WHERE invoice_id = $1 ORDER BY id ASC`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []app.AuditEntry
	for rows.Next() {
		var e app.AuditEntry
		var payloadJSON []byte
		if err := rows.Scan(
			&e.ID, &e.InvoiceID, &e.EventType, &e.Actor,
			&payloadJSON, &e.PrevHash, &e.ChainHash, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(payloadJSON) > 0 {
			_ = json.Unmarshal(payloadJSON, &e.Payload)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
