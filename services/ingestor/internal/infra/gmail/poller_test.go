package gmail_test

import (
	"testing"

	"golang.org/x/oauth2"

	gmailinfra "apex/ingestor/internal/infra/gmail"
)

// TestOAuthConfig_Scopes verifies the OAuth config has the expected Gmail scope.
func TestOAuthConfig_Scopes(t *testing.T) {
	cfg := gmailinfra.OAuthConfig("client-id", "client-secret", "http://localhost/callback")
	if cfg == nil {
		t.Fatal("OAuthConfig returned nil")
	}
	if len(cfg.Scopes) == 0 {
		t.Fatal("expected at least one scope")
	}
	found := false
	for _, s := range cfg.Scopes {
		if s == "https://www.googleapis.com/auth/gmail.readonly" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gmail.readonly scope not found in %v", cfg.Scopes)
	}
}

// TestOAuthConfig_Endpoint verifies the OAuth endpoint is set to Google.
func TestOAuthConfig_Endpoint(t *testing.T) {
	cfg := gmailinfra.OAuthConfig("id", "secret", "uri")
	want := oauth2.Endpoint{
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
	}
	if cfg.Endpoint.AuthURL != want.AuthURL {
		t.Errorf("auth URL: want %s, got %s", want.AuthURL, cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != want.TokenURL {
		t.Errorf("token URL: want %s, got %s", want.TokenURL, cfg.Endpoint.TokenURL)
	}
}
