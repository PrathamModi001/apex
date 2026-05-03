package process_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"apex/event-worker/internal/app/process"
	"apex/event-worker/internal/domain"
)

// ---- mock implementations ----

type mockFileReader struct {
	data        []byte
	contentType string
	err         error
	called      int
}

func (m *mockFileReader) Download(_ context.Context, _ string) ([]byte, string, error) {
	m.called++
	return m.data, m.contentType, m.err
}

type mockOCR struct {
	fields domain.ExtractedFields
	err    error
	called int
}

func (m *mockOCR) Extract(_ context.Context, _ []byte, _ string) (domain.ExtractedFields, error) {
	m.called++
	return m.fields, m.err
}

type mockPOMatcher struct {
	match  domain.POMatch
	err    error
	called int
}

func (m *mockPOMatcher) Match(_ context.Context, _ string, _ float64) (domain.POMatch, error) {
	m.called++
	return m.match, m.err
}

type mockIdempotency struct {
	seen   bool
	err    error
	called int
}

func (m *mockIdempotency) CheckAndMark(_ context.Context, _ string) (bool, error) {
	m.called++
	return m.seen, m.err
}

type mockWriter struct {
	err    error
	called int
}

func (m *mockWriter) Create(_ context.Context, _ domain.ProcessedInvoice) error {
	m.called++
	return m.err
}

type mockPublisher struct {
	err    error
	called int
}

func (m *mockPublisher) Publish(_ context.Context, _ domain.ProcessedInvoice) error {
	m.called++
	return m.err
}

func (m *mockPublisher) Close() error { return nil }

// ---- helpers ----

func buildRaw() domain.RawInvoice {
	return domain.RawInvoice{
		ID:         "inv-001",
		Source:     "email",
		FileKey:    "uploads/inv-001.pdf",
		SHA256:     "abc123",
		Sender:     "vendor@example.com",
		ReceivedAt: time.Now().UTC(),
		Metadata:   map[string]string{},
	}
}

func defaultFields() domain.ExtractedFields {
	return domain.ExtractedFields{
		InvoiceNo:  "INV-2024-001",
		Amount:     1250.00,
		Currency:   "USD",
		DueDate:    "2024-03-15",
		VendorName: "Acme Corp",
	}
}

func defaultPOMatch() domain.POMatch {
	return domain.POMatch{
		POID:       "PO-999",
		Confidence: 0.95,
		Matched:    true,
	}
}

// ---- tests ----

func TestProcessUseCase(t *testing.T) {
	tests := []struct {
		name           string
		idemSeen       bool
		idemErr        error
		readerErr      error
		ocrErr         error
		ocrFields      domain.ExtractedFields
		poMatchErr     error
		writerErr      error
		publishErr     error
		wantErr        bool
		wantDownload   int
		wantPublish    int
		wantWrite      int
	}{
		{
			name:         "happy path: new invoice processes fully",
			idemSeen:     false,
			ocrFields:    defaultFields(),
			wantDownload: 1,
			wantPublish:  1,
			wantWrite:    1,
		},
		{
			name:         "already seen: idempotency returns true → silent skip",
			idemSeen:     true,
			ocrFields:    defaultFields(),
			wantErr:      false,
			wantDownload: 0,
			wantPublish:  0,
			wantWrite:    0,
		},
		{
			name:         "download fails: returns error",
			idemSeen:     false,
			readerErr:    errors.New("minio unavailable"),
			ocrFields:    defaultFields(),
			wantErr:      true,
			wantDownload: 1,
			wantPublish:  0,
			wantWrite:    0,
		},
		{
			name:         "ocr fails: returns error, no write, no publish",
			idemSeen:     false,
			ocrErr:       errors.New("groq api error"),
			ocrFields:    domain.ExtractedFields{},
			wantErr:      true,
			wantDownload: 1,
			wantPublish:  0,
			wantWrite:    0,
		},
		{
			name:         "write fails: returns error, no publish",
			idemSeen:     false,
			ocrFields:    defaultFields(),
			writerErr:    errors.New("postgres error"),
			wantErr:      true,
			wantDownload: 1,
			wantPublish:  0,
			wantWrite:    1,
		},
		{
			name:         "publish fails: returns wrapped error",
			idemSeen:     false,
			ocrFields:    defaultFields(),
			publishErr:   errors.New("kafka unavailable"),
			wantErr:      true,
			wantDownload: 1,
			wantPublish:  1,
			wantWrite:    1,
		},
		{
			name:      "ocr returns zero amount: still processes with amount=0.0",
			idemSeen:  false,
			ocrFields: domain.ExtractedFields{
				InvoiceNo:  "INV-ZERO",
				Amount:     0.0,
				Currency:   "USD",
				DueDate:    "2024-04-01",
				VendorName: "Zero Vendor",
			},
			wantErr:      false,
			wantDownload: 1,
			wantPublish:  1,
			wantWrite:    1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reader := &mockFileReader{data: []byte("pdf-content"), err: tc.readerErr}
			ocr := &mockOCR{fields: tc.ocrFields, err: tc.ocrErr}
			poMatcher := &mockPOMatcher{match: defaultPOMatch(), err: tc.poMatchErr}
			idem := &mockIdempotency{seen: tc.idemSeen, err: tc.idemErr}
			writer := &mockWriter{err: tc.writerErr}
			publisher := &mockPublisher{err: tc.publishErr}

			uc := process.New(reader, ocr, poMatcher, idem, writer, publisher)
			err := uc.Process(context.Background(), buildRaw())

			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if reader.called != tc.wantDownload {
				t.Errorf("Download called %d times, want %d", reader.called, tc.wantDownload)
			}
			if writer.called != tc.wantWrite {
				t.Errorf("Create called %d times, want %d", writer.called, tc.wantWrite)
			}
			if publisher.called != tc.wantPublish {
				t.Errorf("Publish called %d times, want %d", publisher.called, tc.wantPublish)
			}
		})
	}
}
