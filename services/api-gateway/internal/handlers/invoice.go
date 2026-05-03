package handlers

import (
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/app/invoice"
	"apex/api-gateway/internal/domain"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHub extends EventBus with WebSocket connection management.
// The ws.Hub satisfies this interface.
type WSHub interface {
	app.EventBus
	Register(conn *websocket.Conn)
	Unregister(conn *websocket.Conn)
}

// InvoiceHandler groups HTTP handlers for invoice endpoints.
type InvoiceHandler struct {
	uc  *invoice.UseCase
	hub WSHub
}

// NewInvoiceHandler creates an InvoiceHandler.
func NewInvoiceHandler(uc *invoice.UseCase, hub WSHub) *InvoiceHandler {
	return &InvoiceHandler{uc: uc, hub: hub}
}

// invoiceListResponse is the JSON response for GET /invoices.
type invoiceListResponse struct {
	Invoices []invoiceSummaryJSON `json:"invoices"`
	Total    int                  `json:"total"`
}

type invoiceSummaryJSON struct {
	ID         string  `json:"id"`
	Source     string  `json:"source"`
	Status     string  `json:"status"`
	RiskScore  float64 `json:"riskScore"`
	Decision   string  `json:"decision"`
	VendorName string  `json:"vendorName"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	ReceivedAt string  `json:"receivedAt"`
}

type decisionJSON struct {
	InvoiceID      string                   `json:"invoiceId"`
	Decision       string                   `json:"decision"`
	RiskScore      float64                  `json:"riskScore"`
	ReasoningSteps []map[string]interface{} `json:"reasoningSteps"`
	AuditHash      string                   `json:"auditHash"`
	DecidedAt      string                   `json:"decidedAt"`
}

func toInvoiceSummaryJSON(inv domain.InvoiceSummary) invoiceSummaryJSON {
	return invoiceSummaryJSON{
		ID:         inv.ID,
		Source:     inv.Source,
		Status:     inv.Status,
		RiskScore:  inv.RiskScore,
		Decision:   inv.Decision,
		VendorName: inv.VendorName,
		Amount:     inv.Amount,
		Currency:   inv.Currency,
		ReceivedAt: inv.ReceivedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ListInvoices handles GET /invoices.
func (h *InvoiceHandler) ListInvoices(c echo.Context) error {
	filters := app.InvoiceFilters{}
	filters.Status = c.QueryParam("status")
	filters.Source = c.QueryParam("source")

	if v := c.QueryParam("min_risk"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filters.MinRisk = f
		}
	}
	if v := c.QueryParam("max_risk"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			filters.MaxRisk = f
		}
	}
	if v := c.QueryParam("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filters.Page = n
		}
	}
	if v := c.QueryParam("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filters.PageSize = n
		}
	}

	invoices, err := h.uc.List(c.Request().Context(), filters)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	result := make([]invoiceSummaryJSON, 0, len(invoices))
	for _, inv := range invoices {
		result = append(result, toInvoiceSummaryJSON(inv))
	}

	return c.JSON(http.StatusOK, invoiceListResponse{
		Invoices: result,
		Total:    len(result),
	})
}

// GetInvoice handles GET /invoices/:id.
func (h *InvoiceHandler) GetInvoice(c echo.Context) error {
	id := c.Param("id")
	inv, err := h.uc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, toInvoiceSummaryJSON(*inv))
}

// GetDecision handles GET /invoices/:id/decision.
func (h *InvoiceHandler) GetDecision(c echo.Context) error {
	id := c.Param("id")
	d, err := h.uc.GetDecision(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, decisionJSON{
		InvoiceID:      d.InvoiceID,
		Decision:       d.Decision,
		RiskScore:      d.RiskScore,
		ReasoningSteps: d.ReasoningSteps,
		AuditHash:      d.AuditHash,
		DecidedAt:      d.DecidedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ApproveInvoice handles POST /invoices/:id/approve.
func (h *InvoiceHandler) ApproveInvoice(c echo.Context) error {
	id := c.Param("id")
	actor, _ := c.Get("email").(string)
	if actor == "" {
		actor, _ = c.Get("user_id").(string)
	}
	if err := h.uc.Approve(c.Request().Context(), id, actor); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "approved"})
}

// RejectInvoice handles POST /invoices/:id/reject.
func (h *InvoiceHandler) RejectInvoice(c echo.Context) error {
	id := c.Param("id")
	actor, _ := c.Get("email").(string)
	if actor == "" {
		actor, _ = c.Get("user_id").(string)
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := h.uc.Reject(c.Request().Context(), id, actor, body.Reason); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "rejected"})
}

// WSHandler handles GET /ws — upgrades to WebSocket and registers with hub.
func (h *InvoiceHandler) WSHandler(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	h.hub.Register(conn)
	// Read loop: keep the connection alive and handle client-initiated close.
	go func() {
		defer h.hub.Unregister(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return nil
}
