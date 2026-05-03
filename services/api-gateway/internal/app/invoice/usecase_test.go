package invoice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/app/invoice"
	"apex/api-gateway/internal/domain"
)

// --- fake repo ---

type fakeInvoiceRepo struct {
	invoices  map[string]domain.InvoiceSummary
	decisions map[string]domain.Decision
	approved  []string
	rejected  []string
}

func newFakeRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{
		invoices:  make(map[string]domain.InvoiceSummary),
		decisions: make(map[string]domain.Decision),
	}
}

func (r *fakeInvoiceRepo) List(_ context.Context, filters app.InvoiceFilters) ([]domain.InvoiceSummary, error) {
	var out []domain.InvoiceSummary
	for _, inv := range r.invoices {
		if filters.Status != "" && inv.Status != filters.Status {
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

func (r *fakeInvoiceRepo) Get(_ context.Context, id string) (*domain.InvoiceSummary, error) {
	inv, ok := r.invoices[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &inv, nil
}

func (r *fakeInvoiceRepo) GetDecision(_ context.Context, id string) (*domain.Decision, error) {
	d, ok := r.decisions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &d, nil
}

func (r *fakeInvoiceRepo) Approve(_ context.Context, id, actor string) error {
	if _, ok := r.invoices[id]; !ok {
		return errors.New("not found")
	}
	r.approved = append(r.approved, id)
	return nil
}

func (r *fakeInvoiceRepo) Reject(_ context.Context, id, actor, reason string) error {
	if _, ok := r.invoices[id]; !ok {
		return errors.New("not found")
	}
	r.rejected = append(r.rejected, id)
	return nil
}

// --- tests ---

func TestList_ReturnsAll(t *testing.T) {
	repo := newFakeRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", Status: "pending"}
	repo.invoices["inv2"] = domain.InvoiceSummary{ID: "inv2", Status: "approved"}

	uc := invoice.New(repo)
	results, err := uc.List(context.Background(), app.InvoiceFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("want 2 invoices, got %d", len(results))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	repo := newFakeRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", Status: "pending"}
	repo.invoices["inv2"] = domain.InvoiceSummary{ID: "inv2", Status: "approved"}

	uc := invoice.New(repo)
	results, err := uc.List(context.Background(), app.InvoiceFilters{Status: "pending"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(results) != 1 || results[0].Status != "pending" {
		t.Errorf("expected 1 pending invoice, got %d", len(results))
	}
}

func TestGet_Found(t *testing.T) {
	repo := newFakeRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", VendorName: "Acme", Amount: 100.0}

	uc := invoice.New(repo)
	inv, err := uc.Get(context.Background(), "inv1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if inv.VendorName != "Acme" {
		t.Errorf("want vendor=Acme, got %q", inv.VendorName)
	}
}

func TestGet_EmptyID(t *testing.T) {
	uc := invoice.New(newFakeRepo())
	_, err := uc.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestGetDecision_Found(t *testing.T) {
	repo := newFakeRepo()
	repo.decisions["inv1"] = domain.Decision{
		InvoiceID: "inv1",
		Decision:  "approve",
		RiskScore: 0.3,
		DecidedAt: time.Now(),
	}

	uc := invoice.New(repo)
	d, err := uc.GetDecision(context.Background(), "inv1")
	if err != nil {
		t.Fatalf("GetDecision: %v", err)
	}
	if d.Decision != "approve" {
		t.Errorf("want decision=approve, got %q", d.Decision)
	}
}

func TestApprove_OK(t *testing.T) {
	repo := newFakeRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1"}

	uc := invoice.New(repo)
	if err := uc.Approve(context.Background(), "inv1", "user@test.com"); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if len(repo.approved) != 1 || repo.approved[0] != "inv1" {
		t.Error("expected inv1 in approved list")
	}
}

func TestReject_OK(t *testing.T) {
	repo := newFakeRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1"}

	uc := invoice.New(repo)
	if err := uc.Reject(context.Background(), "inv1", "user@test.com", "duplicate"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if len(repo.rejected) != 1 {
		t.Error("expected inv1 in rejected list")
	}
}
