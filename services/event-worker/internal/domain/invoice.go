package domain

import "time"

// RawInvoice — received from invoice.raw topic
type RawInvoice struct {
	ID         string            `json:"id"`
	Source     string            `json:"source"`
	FileKey    string            `json:"file_key"`
	SHA256     string            `json:"sha256"`
	Sender     string            `json:"sender"`
	ReceivedAt time.Time         `json:"received_at"`
	Metadata   map[string]string `json:"metadata"`
}

// ExtractedFields — OCR output
type ExtractedFields struct {
	InvoiceNo  string  `json:"invoice_no"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	DueDate    string  `json:"due_date"`
	VendorName string  `json:"vendor_name"`
}

// POMatch — result of PO matching
type POMatch struct {
	POID       string  `json:"po_id"`
	Confidence float64 `json:"confidence"`
	Matched    bool    `json:"matched"`
}

// ProcessedInvoice — published to invoice.processed
type ProcessedInvoice struct {
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	FileKey         string          `json:"file_key"`
	SHA256          string          `json:"sha256"`
	Sender          string          `json:"sender"`
	ReceivedAt      time.Time       `json:"received_at"`
	ExtractedFields ExtractedFields `json:"extracted_fields"`
	POMatch         POMatch         `json:"po_match"`
	ProcessedAt     time.Time       `json:"processed_at"`
}
