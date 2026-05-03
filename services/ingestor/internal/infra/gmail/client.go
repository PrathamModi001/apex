package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	gmailv1 "google.golang.org/api/gmail/v1"
)

const systemUserID = "00000000-0000-0000-0000-000000000001"

// OAuthConfig returns the oauth2 Config for Gmail read-only access.
func OAuthConfig(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{gmailv1.GmailReadonlyScope},
		Endpoint:     googleoauth.Endpoint,
	}
}

// SaveToken persists (upserts) an OAuth token for the system user.
func SaveToken(ctx context.Context, pool *pgxpool.Pool, tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO google_tokens (user_id, token_json)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET token_json = EXCLUDED.token_json
	`, systemUserID, string(data))
	if err != nil {
		return fmt.Errorf("upsert token: %w", err)
	}
	return nil
}

// LoadToken retrieves the stored OAuth token for the system user.
// Returns nil, nil if no token is stored yet.
func LoadToken(ctx context.Context, pool *pgxpool.Pool) (*oauth2.Token, error) {
	var tokenJSON string
	err := pool.QueryRow(ctx,
		`SELECT token_json FROM google_tokens WHERE user_id = $1`, systemUserID,
	).Scan(&tokenJSON)
	if err != nil {
		// No row found is not an error — just means no token stored yet
		return nil, nil
	}

	var tok oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &tok, nil
}

// TokenSource builds a refreshing TokenSource from the stored token.
func TokenSource(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token) oauth2.TokenSource {
	ts := cfg.TokenSource(ctx, tok)
	log.Println("[gmail] token source initialised")
	return ts
}
