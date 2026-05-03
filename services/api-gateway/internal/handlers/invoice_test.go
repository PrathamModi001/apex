package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/app/invoice"
	"apex/api-gateway/internal/domain"
	"apex/api-gateway/internal/handlers"
)

// --- fakes ---

type fakeInvoiceRepo struct {
	invoices  map[string]domain.InvoiceSummary
	decisions map[string]domain.Decision
	approved  []string
	rejected  []string
}

func newFakeInvoiceRepo() *fakeInvoiceRepo {
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

func (r *fakeInvoiceRepo) Approve(_ context.Context, id, _ string) error {
	if _, ok := r.invoices[id]; !ok {
		return errors.New("not found")
	}
	r.approved = append(r.approved, id)
	return nil
}

func (r *fakeInvoiceRepo) Reject(_ context.Context, id, _, _ string) error {
	if _, ok := r.invoices[id]; !ok {
		return errors.New("not found")
	}
	r.rejected = append(r.rejected, id)
	return nil
}

// fakeHub satisfies handlers.WSHub for tests.
type fakeHub struct {
	registered   []*websocket.Conn
	unregistered []*websocket.Conn
	events       []app.DecisionEvent
}

func (h *fakeHub) Register(conn *websocket.Conn)   { h.registered = append(h.registered, conn) }
func (h *fakeHub) Unregister(conn *websocket.Conn) { h.unregistered = append(h.unregistered, conn) }
func (h *fakeHub) Broadcast(event app.DecisionEvent) {
	h.events = append(h.events, event)
}

// --- helpers ---

func newHandler(repo *fakeInvoiceRepo) (*handlers.InvoiceHandler, *fakeHub) {
	hub := &fakeHub{}
	uc := invoice.New(repo)
	return handlers.NewInvoiceHandler(uc, hub), hub
}

func callHandler(t *testing.T, h echo.HandlerFunc, method, path string, body string, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	for k, v := range params {
		c.SetParamNames(k)
		c.SetParamValues(v)
	}
	if err := h(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

// --- tests ---

func TestListInvoices_ReturnsAll(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", Status: "pending", ReceivedAt: time.Now()}
	repo.invoices["inv2"] = domain.InvoiceSummary{ID: "inv2", Status: "approved", ReceivedAt: time.Now()}

	h, _ := newHandler(repo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListInvoices(c); err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp struct {
		Invoices []map[string]interface{} `json:"invoices"`
		Total    int                      `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("want total=2, got %d", resp.Total)
	}
}

func TestListInvoices_FilterByStatus(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", Status: "pending", ReceivedAt: time.Now()}
	repo.invoices["inv2"] = domain.InvoiceSummary{ID: "inv2", Status: "approved", ReceivedAt: time.Now()}

	h, _ := newHandler(repo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/invoices?status=pending", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.ListInvoices(c); err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}

	var resp struct {
		Invoices []map[string]interface{} `json:"invoices"`
		Total    int                      `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("want total=1, got %d", resp.Total)
	}
}

func TestGetInvoice_Found(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{
		ID: "inv1", VendorName: "Acme", Amount: 999.99, ReceivedAt: time.Now(),
	}

	h, _ := newHandler(repo)
	rec := callHandler(t, h.GetInvoice, http.MethodGet, "/invoices/inv1", "", map[string]string{"id": "inv1"})

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rec.Code, rec.Body)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["vendorName"] != "Acme" {
		t.Errorf("want vendorName=Acme, got %v", resp["vendorName"])
	}
}

func TestGetInvoice_NotFound(t *testing.T) {
	h, _ := newHandler(newFakeInvoiceRepo())
	rec := callHandler(t, h.GetInvoice, http.MethodGet, "/invoices/missing", "", map[string]string{"id": "missing"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", rec.Code)
	}
}

func TestGetDecision_Found(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.decisions["inv1"] = domain.Decision{
		InvoiceID: "inv1", Decision: "approve", RiskScore: 30.0, DecidedAt: time.Now(),
	}

	h, _ := newHandler(repo)
	rec := callHandler(t, h.GetDecision, http.MethodGet, "/invoices/inv1/decision", "", map[string]string{"id": "inv1"})

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rec.Code, rec.Body)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["decision"] != "approve" {
		t.Errorf("want decision=approve, got %v", resp["decision"])
	}
}

func TestApproveInvoice_OK(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", ReceivedAt: time.Now()}

	h, _ := newHandler(repo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/invoices/inv1/approve", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("inv1")
	c.Set("email", "reviewer@test.com")

	if err := h.ApproveInvoice(c); err != nil {
		t.Fatalf("ApproveInvoice: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if len(repo.approved) != 1 || repo.approved[0] != "inv1" {
		t.Error("expected inv1 in approved list")
	}
}

func TestRejectInvoice_OK(t *testing.T) {
	repo := newFakeInvoiceRepo()
	repo.invoices["inv1"] = domain.InvoiceSummary{ID: "inv1", ReceivedAt: time.Now()}

	h, _ := newHandler(repo)

	e := echo.New()
	body := `{"reason":"duplicate invoice"}`
	req := httptest.NewRequest(http.MethodPost, "/invoices/inv1/reject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("inv1")
	c.Set("email", "reviewer@test.com")

	if err := h.RejectInvoice(c); err != nil {
		t.Fatalf("RejectInvoice: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if len(repo.rejected) != 1 {
		t.Error("expected inv1 in rejected list")
	}
}
