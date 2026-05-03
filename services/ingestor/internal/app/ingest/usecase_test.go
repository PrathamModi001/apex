package ingest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/domain"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockStorage struct {
	mu       sync.Mutex
	uploaded []uploadCall
	err      error
}

type uploadCall struct {
	key         string
	contentType string
	data        []byte
}

func (m *mockStorage) Upload(_ context.Context, key, contentType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.uploaded = append(m.uploaded, uploadCall{key, contentType, data})
	return nil
}

type mockDedup struct {
	mu      sync.Mutex
	seen    map[string]struct{}
	err     error
	callLog []string
}

func newMockDedup() *mockDedup {
	return &mockDedup{seen: make(map[string]struct{})}
}

func (m *mockDedup) CheckAndMark(_ context.Context, sha256 string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return false, m.err
	}
	m.callLog = append(m.callLog, sha256)
	if _, ok := m.seen[sha256]; ok {
		return true, nil // duplicate
	}
	m.seen[sha256] = struct{}{}
	return false, nil // new
}

type mockPublisher struct {
	mu        sync.Mutex
	published []domain.RawInvoice
	err       error
}

func (m *mockPublisher) Publish(_ context.Context, inv domain.RawInvoice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, inv)
	return nil
}

func (m *mockPublisher) Close() error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newUC(s *mockStorage, d *mockDedup, p *mockPublisher) *ingest.IngestUseCase {
	return ingest.New(s, d, p)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// 1. Happy path: new invoice → dedup marks → uploads → publishes
func TestIngest_HappyPath(t *testing.T) {
	stor := &mockStorage{}
	dedup := newMockDedup()
	pub := &mockPublisher{}
	uc := newUC(stor, dedup, pub)

	data := []byte("fake PDF content")
	err := uc.Ingest(context.Background(), domain.SourceGmail, "invoice.pdf", data, map[string]string{"email_id": "abc123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stor.uploaded) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(stor.uploaded))
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(pub.published))
	}

	inv := pub.published[0]
	if inv.Source != domain.SourceGmail {
		t.Errorf("want source gmail, got %s", inv.Source)
	}
	if inv.SHA256 == "" {
		t.Error("SHA256 must not be empty")
	}
	if inv.FileKey == "" {
		t.Error("FileKey must not be empty")
	}
	if inv.ID == "" {
		t.Error("ID must not be empty")
	}
}

// 2. Duplicate: CheckAndMark returns (true, nil) → no upload, no publish, returns nil
func TestIngest_Duplicate(t *testing.T) {
	stor := &mockStorage{}
	dedup := newMockDedup()
	pub := &mockPublisher{}
	uc := newUC(stor, dedup, pub)

	data := []byte("repeated invoice")

	// First call — new
	if err := uc.Ingest(context.Background(), domain.SourceGmail, "inv.pdf", data, nil); err != nil {
		t.Fatalf("first ingest error: %v", err)
	}
	// Second call — duplicate
	if err := uc.Ingest(context.Background(), domain.SourceGmail, "inv.pdf", data, nil); err != nil {
		t.Fatalf("second ingest should return nil for duplicate, got: %v", err)
	}

	if len(stor.uploaded) != 1 {
		t.Errorf("expected 1 upload (only first), got %d", len(stor.uploaded))
	}
	if len(pub.published) != 1 {
		t.Errorf("expected 1 published (only first), got %d", len(pub.published))
	}
}

// 3. Storage error: Upload fails → returns error, no publish
func TestIngest_StorageError(t *testing.T) {
	stor := &mockStorage{err: errors.New("minio unavailable")}
	dedup := newMockDedup()
	pub := &mockPublisher{}
	uc := newUC(stor, dedup, pub)

	err := uc.Ingest(context.Background(), domain.SourceTest, "file.pdf", []byte("data"), nil)
	if err == nil {
		t.Fatal("expected error from storage, got nil")
	}
	if len(pub.published) != 0 {
		t.Errorf("expected no publishes after storage error, got %d", len(pub.published))
	}
}

// 4. Publisher error: Publish fails → returns error (file was uploaded)
func TestIngest_PublisherError(t *testing.T) {
	stor := &mockStorage{}
	dedup := newMockDedup()
	pub := &mockPublisher{err: errors.New("kafka unavailable")}
	uc := newUC(stor, dedup, pub)

	err := uc.Ingest(context.Background(), domain.SourceTest, "file.pdf", []byte("some data"), nil)
	if err == nil {
		t.Fatal("expected error from publisher, got nil")
	}
	if len(stor.uploaded) != 1 {
		t.Errorf("file should have been uploaded before publish attempt, got %d uploads", len(stor.uploaded))
	}
}

// 5. Empty data: zero-length data → still processes
func TestIngest_EmptyData(t *testing.T) {
	stor := &mockStorage{}
	dedup := newMockDedup()
	pub := &mockPublisher{}
	uc := newUC(stor, dedup, pub)

	err := uc.Ingest(context.Background(), domain.SourceTest, "empty.pdf", []byte{}, nil)
	if err != nil {
		t.Fatalf("unexpected error for empty data: %v", err)
	}
	if len(stor.uploaded) != 1 {
		t.Errorf("expected 1 upload for empty data, got %d", len(stor.uploaded))
	}
	if len(pub.published) != 1 {
		t.Errorf("expected 1 publish for empty data, got %d", len(pub.published))
	}
}

// 6. Concurrent same SHA256: race condition — only one should succeed
func TestIngest_ConcurrentDuplicate(t *testing.T) {
	stor := &mockStorage{}
	dedup := newMockDedup()
	pub := &mockPublisher{}
	uc := newUC(stor, dedup, pub)

	data := []byte("concurrent invoice data")
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = uc.Ingest(context.Background(), domain.SourceTest, "concurrent.pdf", data, nil)
		}()
	}
	wg.Wait()

	stor.mu.Lock()
	uploads := len(stor.uploaded)
	stor.mu.Unlock()

	pub.mu.Lock()
	publishes := len(pub.published)
	pub.mu.Unlock()

	if uploads != 1 {
		t.Errorf("concurrent: expected exactly 1 upload, got %d", uploads)
	}
	if publishes != 1 {
		t.Errorf("concurrent: expected exactly 1 publish, got %d", publishes)
	}
}
