package telegram_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/domain"
	telegraminfra "apex/ingestor/internal/infra/telegram"
)

// ---------------------------------------------------------------------------
// Mocks for use-case dependencies
// ---------------------------------------------------------------------------

type mockStorageTG struct {
	mu       sync.Mutex
	uploaded int
}

func (m *mockStorageTG) Upload(_ context.Context, _, _ string, _ []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploaded++
	return nil
}

type mockDedupTG struct{}

func (m *mockDedupTG) CheckAndMark(_ context.Context, _ string) (bool, error) {
	return false, nil // always new
}

type mockPublisherTG struct {
	mu        sync.Mutex
	published []domain.RawInvoice
}

func (m *mockPublisherTG) Publish(_ context.Context, inv domain.RawInvoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, inv)
	return nil
}

func (m *mockPublisherTG) Close() error { return nil }

// ---------------------------------------------------------------------------
// Telegram fake server
// ---------------------------------------------------------------------------

// newTelegramFakeServer returns an httptest.Server that handles getFile and
// file-download requests, plus the URL to use as bot-api base in tests.
func newTelegramFakeServer(t *testing.T, fileContent []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/bot", func(w http.ResponseWriter, r *http.Request) {
		// getFile
		resp := map[string]interface{}{
			"ok": true,
			"result": map[string]string{
				"file_id":   "file123",
				"file_path": "documents/invoice.pdf",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/file/bot", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(fileContent)
	})
	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testSecret = "test-webhook-secret"

func newTestHandler() (*telegraminfra.WebhookHandler, *mockPublisherTG, *mockStorageTG) {
	stor := &mockStorageTG{}
	dedup := &mockDedupTG{}
	pub := &mockPublisherTG{}
	uc := ingest.New(stor, dedup, pub)
	h := telegraminfra.NewWebhookHandler(uc, "BOTTOKEN", testSecret)
	return h, pub, stor
}

func buildUpdateJSON(t *testing.T, doc *telegraminfra.Document, photos []telegraminfra.PhotoSize) []byte {
	t.Helper()
	msg := map[string]interface{}{
		"message_id": 1,
		"from":       map[string]interface{}{"id": 42, "username": "testuser"},
	}
	if doc != nil {
		msg["document"] = doc
	}
	if photos != nil {
		msg["photo"] = photos
	}
	update := map[string]interface{}{
		"update_id": 100,
		"message":   msg,
	}
	b, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal update: %v", err)
	}
	return b
}

func doRequest(t *testing.T, h *telegraminfra.WebhookHandler, body []byte, secretHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secretHeader != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretHeader)
	}
	rec := httptest.NewRecorder()
	h.Handle(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// 1. Valid secret token → handler accepts the request
func TestWebhook_ValidSecret(t *testing.T) {
	h, _, _ := newTestHandler()
	body := buildUpdateJSON(t, nil, nil) // text-only message
	rec := doRequest(t, h, body, testSecret)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

// 2. Wrong secret token → 401
func TestWebhook_InvalidSecret(t *testing.T) {
	h, _, _ := newTestHandler()
	body := buildUpdateJSON(t, nil, nil)
	rec := doRequest(t, h, body, "wrong-secret")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

// 3. Missing secret token → 401
func TestWebhook_MissingSecret(t *testing.T) {
	h, _, _ := newTestHandler()
	body := buildUpdateJSON(t, nil, nil)
	rec := doRequest(t, h, body, "") // no header
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

// 4. Text-only message (no attachment) → 200, no ingest
func TestWebhook_TextOnly(t *testing.T) {
	h, pub, stor := newTestHandler()
	body := buildUpdateJSON(t, nil, nil)
	rec := doRequest(t, h, body, testSecret)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.published) != 0 {
		t.Errorf("expected 0 publishes for text-only message, got %d", len(pub.published))
	}
	stor.mu.Lock()
	defer stor.mu.Unlock()
	if stor.uploaded != 0 {
		t.Errorf("expected 0 uploads for text-only message, got %d", stor.uploaded)
	}
}

// 5. Document attachment → ingest is called
// Since we don't want to spin up a real Telegram API server, we verify the
// handler calls the right code path by injecting a custom http.Client via a
// test server that serves a fake getFile and file-download response.
func TestWebhook_DocumentAttachment(t *testing.T) {
	fakeContent := []byte("%PDF-1.4 fake content")

	// Build a fake Telegram API server
	telegramSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// getFile endpoint: /botTOKEN/getFile
		if r.URL.Path != "" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"result": map[string]string{
					"file_id":   "file123",
					"file_path": "documents/invoice.pdf",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer telegramSrv.Close()

	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(fakeContent)
	}))
	defer fileSrv.Close()

	stor := &mockStorageTG{}
	dedup := &mockDedupTG{}
	pub := &mockPublisherTG{}
	uc := ingest.New(stor, dedup, pub)

	// Use the test-server-aware handler variant
	h := telegraminfra.NewWebhookHandlerWithClient(
		uc,
		"BOTTOKEN",
		testSecret,
		telegramSrv.URL,
		fileSrv.URL,
		telegramSrv.Client(),
	)

	doc := &telegraminfra.Document{
		FileID:   "file123",
		FileName: "invoice.pdf",
		MimeType: "application/pdf",
	}
	body := buildUpdateJSON(t, doc, nil)
	rec := doRequest(t, h, body, testSecret)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.published) != 1 {
		t.Errorf("expected 1 publish for document, got %d", len(pub.published))
	}
	if pub.published[0].Source != domain.SourceTelegram {
		t.Errorf("want source telegram, got %s", pub.published[0].Source)
	}
}
