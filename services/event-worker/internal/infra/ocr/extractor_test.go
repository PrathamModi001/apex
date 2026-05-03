package ocr_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	ocrinfra "apex/event-worker/internal/infra/ocr"
)

// mockHTTPClient satisfies ocr.HTTPClient for tests.
type mockHTTPClient struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func makeResp(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

const validGroqResp = `{
	"choices": [{
		"message": {
			"content": "{\"invoice_no\":\"INV-2024-001\",\"amount\":1250.00,\"currency\":\"USD\",\"due_date\":\"2024-03-15\",\"vendor_name\":\"Acme Corp\"}"
		}
	}]
}`

func TestExtractor_ValidGroqResponse(t *testing.T) {
	client := &mockHTTPClient{resp: makeResp(200, validGroqResp)}
	ext := ocrinfra.NewWithClient("test-key", client)

	fields, err := ext.Extract(context.Background(), []byte("pdf-data"), "invoice.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields.InvoiceNo != "INV-2024-001" {
		t.Errorf("InvoiceNo = %q, want %q", fields.InvoiceNo, "INV-2024-001")
	}
	if fields.Amount != 1250.00 {
		t.Errorf("Amount = %f, want 1250.00", fields.Amount)
	}
	if fields.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", fields.Currency)
	}
	if fields.VendorName != "Acme Corp" {
		t.Errorf("VendorName = %q, want Acme Corp", fields.VendorName)
	}
}

func TestExtractor_MalformedJSON_ReturnsError(t *testing.T) {
	body := `{"choices":[{"message":{"content":"not-valid-json"}}]}`
	client := &mockHTTPClient{resp: makeResp(200, body)}
	ext := ocrinfra.NewWithClient("test-key", client)

	_, err := ext.Extract(context.Background(), []byte("data"), "inv.pdf")
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestExtractor_HTTP500_ReturnsError(t *testing.T) {
	client := &mockHTTPClient{resp: makeResp(500, `{"error":{"message":"internal server error"}}`)}
	ext := ocrinfra.NewWithClient("test-key", client)

	_, err := ext.Extract(context.Background(), []byte("data"), "inv.pdf")
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code 500, got: %v", err)
	}
}

func TestExtractor_EmptyResponseContent_ReturnsError(t *testing.T) {
	body := `{"choices":[{"message":{"content":""}}]}`
	client := &mockHTTPClient{resp: makeResp(200, body)}
	ext := ocrinfra.NewWithClient("test-key", client)

	_, err := ext.Extract(context.Background(), []byte("data"), "inv.pdf")
	if err == nil {
		t.Error("expected error for empty response content, got nil")
	}
}

func TestExtractor_AmountAsString_ParsesCorrectly(t *testing.T) {
	// Some invoices return amount as "1,250.00" (string with thousands separator)
	body := `{"choices":[{"message":{"content":"{\"invoice_no\":\"INV-STR\",\"amount\":\"1,250.00\",\"currency\":\"EUR\",\"due_date\":\"2024-05-01\",\"vendor_name\":\"String Vendor\"}"}}]}`
	client := &mockHTTPClient{resp: makeResp(200, body)}
	ext := ocrinfra.NewWithClient("test-key", client)

	fields, err := ext.Extract(context.Background(), []byte("data"), "inv.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields.Amount != 1250.00 {
		t.Errorf("Amount = %f, want 1250.00", fields.Amount)
	}
	if fields.InvoiceNo != "INV-STR" {
		t.Errorf("InvoiceNo = %q, want INV-STR", fields.InvoiceNo)
	}
}
