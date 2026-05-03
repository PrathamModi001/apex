package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Notifier implements app.TelegramNotifier by calling the Telegram Bot API.
type Notifier struct {
	botToken string
	chatID   string
}

// NewNotifier creates a Notifier for the given bot token and admin chat ID.
func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{botToken: botToken, chatID: chatID}
}

// SendApprovalRequest formats and sends an approval-required message to the admin chat.
// The chatID parameter is accepted for interface compatibility but the notifier uses its
// own configured chatID (TELEGRAM_ADMIN_CHAT_ID) to route the message.
func (n *Notifier) SendApprovalRequest(chatID, invoiceID, vendor string, amount float64, riskScore float64, reason string) error {
	effectiveChatID := chatID
	if effectiveChatID == "" {
		effectiveChatID = n.chatID
	}

	text := fmt.Sprintf(
		"⚠️ Invoice Review Required\nVendor: %s\nAmount: %.2f\nRisk: %.0f/100\nID: %s\nReason: %s",
		vendor, amount, riskScore, invoiceID, reason,
	)

	payload := map[string]string{
		"chat_id": effectiveChatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return fmt.Errorf("telegram: http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram: API returned status %d", resp.StatusCode)
	}
	return nil
}
