package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/domain"
)

// InvoiceRepo implements app.InvoiceRepository using pgx v5.
type InvoiceRepo struct {
	pool *pgxpool.Pool
}

// NewInvoiceRepo creates an InvoiceRepo backed by the given pool.
func NewInvoiceRepo(pool *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{pool: pool}
}

// List returns invoices matching the provided filters.
func (r *InvoiceRepo) List(ctx context.Context, filters app.InvoiceFilters) ([]domain.InvoiceSummary, error) {
	query := `SELECT id, source, status, risk_score, decision, vendor_name, amount, currency, received_at
	          FROM invoices WHERE 1=1`
	args := []interface{}{}
	n := 1

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, filters.Status)
		n++
	}
	if filters.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", n)
		args = append(args, filters.Source)
		n++
	}
	if filters.MinRisk > 0 {
		query += fmt.Sprintf(" AND risk_score >= $%d", n)
		args = append(args, filters.MinRisk)
		n++
	}
	if filters.MaxRisk > 0 {
		query += fmt.Sprintf(" AND risk_score <= $%d", n)
		args = append(args, filters.MaxRisk)
		n++
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY received_at DESC LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.InvoiceSummary
	for rows.Next() {
		var inv domain.InvoiceSummary
		if err := rows.Scan(
			&inv.ID, &inv.Source, &inv.Status, &inv.RiskScore,
			&inv.Decision, &inv.VendorName, &inv.Amount, &inv.Currency, &inv.ReceivedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, inv)
	}
	return results, rows.Err()
}

// Get returns a single invoice by ID.
func (r *InvoiceRepo) Get(ctx context.Context, id string) (*domain.InvoiceSummary, error) {
	var inv domain.InvoiceSummary
	err := r.pool.QueryRow(ctx,
		`SELECT id, source, status, risk_score, decision, vendor_name, amount, currency, received_at
		 FROM invoices WHERE id = $1`, id,
	).Scan(
		&inv.ID, &inv.Source, &inv.Status, &inv.RiskScore,
		&inv.Decision, &inv.VendorName, &inv.Amount, &inv.Currency, &inv.ReceivedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetDecision returns the agent decision for an invoice.
func (r *InvoiceRepo) GetDecision(ctx context.Context, id string) (*domain.Decision, error) {
	var d domain.Decision
	var stepsJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT invoice_id, decision, risk_score, reasoning_steps, audit_hash, decided_at
		 FROM invoice_decisions WHERE invoice_id = $1`, id,
	).Scan(&d.InvoiceID, &d.Decision, &d.RiskScore, &stepsJSON, &d.AuditHash, &d.DecidedAt)
	if err != nil {
		return nil, err
	}
	if len(stepsJSON) > 0 {
		_ = json.Unmarshal(stepsJSON, &d.ReasoningSteps)
	}
	return &d, nil
}

// Approve marks an invoice as approved.
func (r *InvoiceRepo) Approve(ctx context.Context, id, actor string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invoices SET status='approved', approved_by=$1, approved_at=NOW() WHERE id=$2`,
		actor, id)
	return err
}

// Reject marks an invoice as rejected with a reason.
func (r *InvoiceRepo) Reject(ctx context.Context, id, actor, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invoices SET status='rejected', rejected_by=$1, reject_reason=$2, rejected_at=NOW() WHERE id=$3`,
		actor, reason, id)
	return err
}
