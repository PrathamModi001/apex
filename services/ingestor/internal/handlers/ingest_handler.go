package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/domain"
	gmailinfra "apex/ingestor/internal/infra/gmail"
	telegraminfra "apex/ingestor/internal/infra/telegram"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Google OAuth handlers
// ---------------------------------------------------------------------------

// GoogleAuthHandler redirects the user to Google's OAuth2 consent screen.
func GoogleAuthHandler(cfg *oauth2.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		state := "apex-ingestor-state" // In Phase 6 use a CSRF token per-user
		url := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
		return c.Redirect(http.StatusFound, url)
	}
}

// GoogleCallbackHandler exchanges the OAuth code for a token and stores it.
func GoogleCallbackHandler(cfg *oauth2.Config, pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.QueryParam("code")
		if code == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing code"})
		}

		ctx := c.Request().Context()
		tok, err := cfg.Exchange(ctx, code)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if err := gmailinfra.SaveToken(ctx, pool, tok); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "gmail connected"})
	}
}

// ---------------------------------------------------------------------------
// Telegram webhook handler (adapter from echo.HandlerFunc to net/http)
// ---------------------------------------------------------------------------

// TelegramWebhookHandler wraps the telegram WebhookHandler for use with Echo.
func TelegramWebhookHandler(wh *telegraminfra.WebhookHandler) echo.HandlerFunc {
	return func(c echo.Context) error {
		wh.Handle(c.Response().Writer, c.Request())
		return nil
	}
}

// ---------------------------------------------------------------------------
// Test ingest endpoint
// ---------------------------------------------------------------------------

// TestIngestRequest is the body accepted by POST /ingest/test
type TestIngestRequest struct {
	Vendor   string  `json:"vendor"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// TestIngestHandler runs a synthetic invoice through the full ingest pipeline.
func TestIngestHandler(uc *ingest.IngestUseCase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req TestIngestRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		}

		data, err := json.Marshal(req)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		metadata := map[string]string{
			"vendor":   req.Vendor,
			"amount":   amountToString(req.Amount),
			"currency": req.Currency,
		}

		ctx := c.Request().Context()
		if err := uc.Ingest(ctx, domain.SourceTest, "test-invoice.json", data, metadata); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusAccepted, map[string]string{"status": "ingested"})
	}
}

func amountToString(f float64) string {
	return json.Number(formatFloat(f)).String()
}

func formatFloat(f float64) string {
	// Use strconv for clean float formatting
	b, _ := json.Marshal(f)
	return string(b)
}
