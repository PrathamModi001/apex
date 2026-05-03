package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"apex/event-worker/internal/domain"
)

const (
	groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"
	groqModel    = "llama-3.3-70b-versatile"
	httpTimeout  = 30 * time.Second
)

// HTTPClient is an interface for making HTTP requests, enabling test injection.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Extractor calls the Groq API to extract invoice fields from raw document bytes.
type Extractor struct {
	apiKey     string
	httpClient HTTPClient
}

// New creates an Extractor with the given API key.
// If apiKey is empty, stub fields are returned (for local dev without API key).
func New(apiKey string) *Extractor {
	return &Extractor{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: httpTimeout},
	}
}

// NewWithClient creates an Extractor with a custom HTTP client (for testing).
func NewWithClient(apiKey string, httpClient HTTPClient) *Extractor {
	return &Extractor{
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// groqRequest is the request body sent to the Groq chat completions API.
type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

// groqMessage is a single message in the chat completions request.
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqResponse is the response from the Groq chat completions API.
type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// rawExtractedFields is used for flexible JSON parsing (amount may be string or number).
type rawExtractedFields struct {
	InvoiceNo  string      `json:"invoice_no"`
	Amount     interface{} `json:"amount"`
	Currency   string      `json:"currency"`
	DueDate    string      `json:"due_date"`
	VendorName string      `json:"vendor_name"`
}

// Extract sends the document bytes to Groq and parses the structured invoice fields.
func (e *Extractor) Extract(ctx context.Context, data []byte, filename string) (domain.ExtractedFields, error) {
	// If no API key, return stub fields for local dev
	if e.apiKey == "" {
		return domain.ExtractedFields{
			InvoiceNo:  "STUB-001",
			Amount:     0.0,
			Currency:   "USD",
			DueDate:    "2024-01-01",
			VendorName: "Stub Vendor",
		}, nil
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	prompt := fmt.Sprintf(
		`You are an invoice parser. Given this base64-encoded document content, extract: invoice_no, amount (float), currency, due_date (YYYY-MM-DD), vendor_name. Respond with ONLY valid JSON: {"invoice_no":"...","amount":0.0,"currency":"USD","due_date":"...","vendor_name":"..."}\n\n%s`,
		encoded,
	)

	reqBody := groqRequest{
		Model: groqModel,
		Messages: []groqMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("marshal groq request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("build groq request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("groq http call: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("read groq response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return domain.ExtractedFields{}, fmt.Errorf("groq returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var groqResp groqResponse
	if err := json.Unmarshal(respBytes, &groqResp); err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("unmarshal groq response: %w", err)
	}

	if len(groqResp.Choices) == 0 || groqResp.Choices[0].Message.Content == "" {
		return domain.ExtractedFields{}, fmt.Errorf("groq response has no content")
	}

	content := strings.TrimSpace(groqResp.Choices[0].Message.Content)

	// Strip markdown code blocks if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var raw rawExtractedFields
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("parse extracted fields JSON: %w", err)
	}

	amount, err := parseAmount(raw.Amount)
	if err != nil {
		return domain.ExtractedFields{}, fmt.Errorf("parse amount: %w", err)
	}

	return domain.ExtractedFields{
		InvoiceNo:  raw.InvoiceNo,
		Amount:     amount,
		Currency:   raw.Currency,
		DueDate:    raw.DueDate,
		VendorName: raw.VendorName,
	}, nil
}

// parseAmount handles amount values that may be float64 or string (e.g. "1,250.00").
func parseAmount(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case string:
		// Remove thousands separators and parse
		cleaned := strings.ReplaceAll(val, ",", "")
		cleaned = strings.TrimSpace(cleaned)
		f, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return 0, fmt.Errorf("parse amount string %q: %w", val, err)
		}
		return f, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unexpected amount type %T", v)
	}
}
