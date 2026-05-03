package invoice

import (
	"context"
	"fmt"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/domain"
)

// UseCase handles invoice-related business logic.
type UseCase struct {
	repo app.InvoiceRepository
}

// New creates an InvoiceUseCase.
func New(repo app.InvoiceRepository) *UseCase {
	return &UseCase{repo: repo}
}

// List returns invoices matching the provided filters.
func (uc *UseCase) List(ctx context.Context, filters app.InvoiceFilters) ([]domain.InvoiceSummary, error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 {
		filters.PageSize = 20
	}
	return uc.repo.List(ctx, filters)
}

// Get returns a single invoice by ID.
func (uc *UseCase) Get(ctx context.Context, id string) (*domain.InvoiceSummary, error) {
	if id == "" {
		return nil, fmt.Errorf("invoice id is required")
	}
	return uc.repo.Get(ctx, id)
}

// GetDecision returns the agent decision for an invoice.
func (uc *UseCase) GetDecision(ctx context.Context, id string) (*domain.Decision, error) {
	if id == "" {
		return nil, fmt.Errorf("invoice id is required")
	}
	return uc.repo.GetDecision(ctx, id)
}

// Approve marks an invoice as approved by a human actor.
func (uc *UseCase) Approve(ctx context.Context, id, actor string) error {
	if id == "" {
		return fmt.Errorf("invoice id is required")
	}
	if actor == "" {
		return fmt.Errorf("actor is required")
	}
	return uc.repo.Approve(ctx, id, actor)
}

// Reject marks an invoice as rejected by a human actor with a reason.
func (uc *UseCase) Reject(ctx context.Context, id, actor, reason string) error {
	if id == "" {
		return fmt.Errorf("invoice id is required")
	}
	if actor == "" {
		return fmt.Errorf("actor is required")
	}
	return uc.repo.Reject(ctx, id, actor, reason)
}
