package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/domain"
)

// ---------------------------------------------------------------------------
// Telegram Update types (minimal subset)
// ---------------------------------------------------------------------------

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int      `json:"message_id"`
	From      *User    `json:"from"`
	Document  *Document `json:"document"`
	Photo     []PhotoSize `json:"photo"`
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size"`
}

type getFileResponse struct {
	OK     bool     `json:"ok"`
	Result FileInfo `json:"result"`
}

type FileInfo struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
}

// ---------------------------------------------------------------------------
// WebhookHandler
// ---------------------------------------------------------------------------

// WebhookHandler validates Telegram webhook requests and triggers ingestion.
type WebhookHandler struct {
	uc            *ingest.IngestUseCase
	botToken      string
	webhookSecret string
	httpClient    *http.Client
	// Override base URLs — used in tests to point at fake servers.
	apiBaseURL  string // default: "https://api.telegram.org"
	fileBaseURL string // default: "https://api.telegram.org"
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(uc *ingest.IngestUseCase, botToken, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		uc:            uc,
		botToken:      botToken,
		webhookSecret: webhookSecret,
		httpClient:    &http.Client{},
		apiBaseURL:    "https://api.telegram.org",
		fileBaseURL:   "https://api.telegram.org",
	}
}

// NewWebhookHandlerWithClient creates a WebhookHandler with custom HTTP client
// and base URLs — intended for testing only.
func NewWebhookHandlerWithClient(
	uc *ingest.IngestUseCase,
	botToken, webhookSecret string,
	apiBaseURL, fileBaseURL string,
	client *http.Client,
) *WebhookHandler {
	if client == nil {
		client = &http.Client{}
	}
	return &WebhookHandler{
		uc:            uc,
		botToken:      botToken,
		webhookSecret: webhookSecret,
		httpClient:    client,
		apiBaseURL:    apiBaseURL,
		fileBaseURL:   fileBaseURL,
	}
}

// Handle processes an incoming Telegram webhook HTTP request.
// It validates the X-Telegram-Bot-Api-Secret-Token header, then processes any
// attached document or the largest photo in the message.
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Validate secret token
	secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if secret == "" || secret != h.webhookSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body error", http.StatusBadRequest)
		return
	}

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if update.Message == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	sender := h.extractSender(update.Message)

	// Process document attachment
	if update.Message.Document != nil {
		doc := update.Message.Document
		filename := doc.FileName
		if filename == "" {
			filename = doc.FileID
		}
		if err := h.ingestFile(ctx, doc.FileID, filename, doc.MimeType, sender); err != nil {
			log.Printf("[telegram] ingest document: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Process largest photo (last element has the highest resolution)
	if len(update.Message.Photo) > 0 {
		photo := update.Message.Photo[len(update.Message.Photo)-1]
		if err := h.ingestFile(ctx, photo.FileID, photo.FileID+".jpg", "image/jpeg", sender); err != nil {
			log.Printf("[telegram] ingest photo: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Text-only message — no-op
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) ingestFile(ctx context.Context, fileID, filename, mimeType, sender string) error {
	filePath, err := h.getFilePath(ctx, fileID)
	if err != nil {
		return fmt.Errorf("getFilePath: %w", err)
	}

	data, err := h.downloadFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("downloadFile: %w", err)
	}

	metadata := map[string]string{
		"sender":       sender,
		"file_id":      fileID,
		"mime_type":    mimeType,
	}

	return h.uc.Ingest(ctx, domain.SourceTelegram, filename, data, metadata)
}

// getFilePath calls the Telegram getFile API to resolve a file_id into a path.
func (h *WebhookHandler) getFilePath(ctx context.Context, fileID string) (string, error) {
	apiURL := fmt.Sprintf("%s/bot%s/getFile?file_id=%s", h.apiBaseURL, h.botToken, fileID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result getFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("telegram getFile returned ok=false")
	}
	return result.Result.FilePath, nil
}

// downloadFile downloads the file from Telegram's CDN.
func (h *WebhookHandler) downloadFile(ctx context.Context, filePath string) ([]byte, error) {
	fileURL := fmt.Sprintf("%s/file/bot%s/%s", h.fileBaseURL, h.botToken, filePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (h *WebhookHandler) extractSender(msg *Message) string {
	if msg.From == nil {
		return ""
	}
	if msg.From.Username != "" {
		return msg.From.Username
	}
	return strconv.FormatInt(msg.From.ID, 10)
}
