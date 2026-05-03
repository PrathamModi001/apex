package gmail

import (
	"context"
	"encoding/base64"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	gmailv1 "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"apex/ingestor/internal/app/ingest"
	"apex/ingestor/internal/domain"
)

const pollInterval = 60 * time.Second

// GmailPoller polls the Gmail AP-Inbox label every 60 seconds.
type GmailPoller struct {
	uc          *ingest.IngestUseCase
	pool        *pgxpool.Pool
	oauthConfig *oauth2.Config
}

// NewPoller creates a GmailPoller.
func NewPoller(uc *ingest.IngestUseCase, pool *pgxpool.Pool, cfg *oauth2.Config) *GmailPoller {
	return &GmailPoller{uc: uc, pool: pool, oauthConfig: cfg}
}

// Start launches the polling loop in a goroutine.
// It returns immediately; cancel the supplied context to stop polling.
func (p *GmailPoller) Start(ctx context.Context) {
	go p.run(ctx)
}

func (p *GmailPoller) run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Poll immediately on first start, then every tick.
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[gmail-poller] stopping")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *GmailPoller) poll(ctx context.Context) {
	tok, err := LoadToken(ctx, p.pool)
	if err != nil {
		log.Printf("[gmail-poller] load token error: %v", err)
		return
	}
	if tok == nil {
		log.Println("[gmail-poller] no Gmail token stored — skipping poll")
		return
	}

	ts := TokenSource(ctx, p.oauthConfig, tok)
	svc, err := gmailv1.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		log.Printf("[gmail-poller] create gmail service: %v", err)
		return
	}

	r, err := svc.Users.Messages.List("me").
		LabelIds("AP-Inbox").
		Q("is:unread has:attachment").
		Do()
	if err != nil {
		log.Printf("[gmail-poller] list messages: %v", err)
		return
	}
	if len(r.Messages) == 0 {
		return
	}

	log.Printf("[gmail-poller] found %d unread messages with attachments", len(r.Messages))

	for _, m := range r.Messages {
		if err := p.processMessage(ctx, svc, m.Id); err != nil {
			log.Printf("[gmail-poller] process message %s: %v", m.Id, err)
		}
	}
}

func (p *GmailPoller) processMessage(ctx context.Context, svc *gmailv1.Service, msgID string) error {
	msg, err := svc.Users.Messages.Get("me", msgID).Format("full").Do()
	if err != nil {
		return err
	}

	sender := extractSender(msg)

	for _, part := range msg.Payload.Parts {
		if part.Filename == "" || part.Body == nil || part.Body.AttachmentId == "" {
			continue
		}

		att, err := svc.Users.Messages.Attachments.Get("me", msgID, part.Body.AttachmentId).Do()
		if err != nil {
			log.Printf("[gmail-poller] get attachment %s: %v", part.Body.AttachmentId, err)
			continue
		}

		data, err := base64.URLEncoding.DecodeString(att.Data)
		if err != nil {
			log.Printf("[gmail-poller] decode attachment data: %v", err)
			continue
		}

		metadata := map[string]string{
			"email_id":     msgID,
			"sender":       sender,
			"filename":     part.Filename,
			"content_type": part.MimeType,
		}

		if err := p.uc.Ingest(ctx, domain.SourceGmail, part.Filename, data, metadata); err != nil {
			log.Printf("[gmail-poller] ingest %s: %v", part.Filename, err)
		}
	}

	// Mark as read by removing the UNREAD label
	_, err = svc.Users.Messages.Modify("me", msgID, &gmailv1.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Do()
	return err
}

// extractSender tries to get the From header value from the message.
func extractSender(msg *gmailv1.Message) string {
	if msg.Payload == nil {
		return ""
	}
	for _, h := range msg.Payload.Headers {
		if h.Name == "From" {
			return h.Value
		}
	}
	return ""
}
