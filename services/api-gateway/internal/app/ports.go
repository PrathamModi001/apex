package app

import (
	"context"
	"time"

	"apex/api-gateway/internal/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	FindOrCreate(ctx context.Context, email, name string) (domain.User, error)
	UpdateRole(ctx context.Context, userID string, role domain.Role) error
	List(ctx context.Context) ([]domain.User, error)
}

// AuditEntry is a single row from the audit_log table.
type AuditEntry struct {
	ID        int64
	InvoiceID string
	EventType string
	Actor     string
	Payload   map[string]interface{}
	PrevHash  string
	ChainHash string
	CreatedAt time.Time
}

// AuditRepository reads the append-only audit log.
type AuditRepository interface {
	GetChain(ctx context.Context, invoiceID string) ([]AuditEntry, error)
}

// InvoiceFilters holds optional filters for listing invoices.
type InvoiceFilters struct {
	Status   string
	Source   string
	MinRisk  float64
	MaxRisk  float64
	Page     int
	PageSize int
}

// InvoiceRepository defines persistence operations for invoices.
type InvoiceRepository interface {
	List(ctx context.Context, filters InvoiceFilters) ([]domain.InvoiceSummary, error)
	Get(ctx context.Context, id string) (*domain.InvoiceSummary, error)
	GetDecision(ctx context.Context, id string) (*domain.Decision, error)
	Approve(ctx context.Context, id string, actor string) error
	Reject(ctx context.Context, id string, actor string, reason string) error
}

// RateLimiter checks and enforces rate limits using a backend store.
type RateLimiter interface {
	Allow(ctx context.Context, key string, rate int, window time.Duration) (bool, error)
}

// TelegramNotifier sends approval request messages via Telegram.
type TelegramNotifier interface {
	SendApprovalRequest(chatID, invoiceID, vendor string, amount float64, riskScore float64, reason string) error
}

// DecisionEvent is the broadcast payload sent to WebSocket clients.
type DecisionEvent struct {
	InvoiceID  string  `json:"invoice_id"`
	Decision   string  `json:"decision"`
	RiskScore  float64 `json:"risk_score"`
	AuditHash  string  `json:"audit_hash"`
	VendorName string  `json:"vendor_name"`
}

// EventBus broadcasts decision events to connected WebSocket clients.
type EventBus interface {
	Broadcast(event DecisionEvent)
}
