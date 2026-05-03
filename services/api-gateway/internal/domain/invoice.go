package domain

import "time"

// InvoiceSummary holds a summary record of an invoice.
type InvoiceSummary struct {
	ID         string
	Source     string
	Status     string
	RiskScore  float64
	Decision   string
	VendorName string
	Amount     float64
	Currency   string
	ReceivedAt time.Time
}

// Decision holds the agent decision for an invoice.
type Decision struct {
	InvoiceID      string
	Decision       string
	RiskScore      float64
	ReasoningSteps []map[string]interface{}
	AuditHash      string
	DecidedAt      time.Time
}
