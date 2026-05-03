package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/domain"
)

// Claims holds JWT payload for RS256 or HMAC tokens.
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthUseCase handles Google OAuth exchange, JWT issuance, and verification.
type AuthUseCase struct {
	userRepo   app.UserRepository
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	hmacSecret []byte      // fallback for local dev
	useHMAC    bool
	oauthCfg   *oauth2.Config
}

// New creates an AuthUseCase.  If RSA keys are not provided it falls back to HMAC using JWT_SECRET.
func New(userRepo app.UserRepository) *AuthUseCase {
	uc := &AuthUseCase{userRepo: userRepo}

	privB64 := os.Getenv("JWT_PRIVATE_KEY")
	pubB64 := os.Getenv("JWT_PUBLIC_KEY")

	if privB64 != "" && pubB64 != "" {
		privPEM, err := base64.StdEncoding.DecodeString(privB64)
		if err != nil {
			log.Printf("auth: failed to base64-decode JWT_PRIVATE_KEY: %v", err)
		}
		pubPEM, err := base64.StdEncoding.DecodeString(pubB64)
		if err != nil {
			log.Printf("auth: failed to base64-decode JWT_PUBLIC_KEY: %v", err)
		}
		if privPEM != nil && pubPEM != nil {
			priv, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
			if err != nil {
				log.Printf("auth: failed to parse RSA private key: %v", err)
			}
			pub, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
			if err != nil {
				log.Printf("auth: failed to parse RSA public key: %v", err)
			}
			if priv != nil && pub != nil {
				uc.privateKey = priv
				uc.publicKey = pub
			}
		}
	}

	if uc.privateKey == nil {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "dev-secret-change-me"
		}
		uc.hmacSecret = []byte(secret)
		uc.useHMAC = true
		log.Println("auth: RSA keys not set — using HMAC JWT_SECRET for local dev")
	}

	uc.oauthCfg = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URI"),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return uc
}

// NewWithKeys creates an AuthUseCase with explicit RSA keys (used in tests).
func NewWithKeys(userRepo app.UserRepository, priv *rsa.PrivateKey, pub *rsa.PublicKey) *AuthUseCase {
	return &AuthUseCase{
		userRepo:   userRepo,
		privateKey: priv,
		publicKey:  pub,
		useHMAC:    false,
		oauthCfg:   &oauth2.Config{},
	}
}

// NewWithHMAC creates an AuthUseCase using HMAC secret (used in tests).
func NewWithHMAC(userRepo app.UserRepository, secret []byte) *AuthUseCase {
	return &AuthUseCase{
		userRepo:   userRepo,
		hmacSecret: secret,
		useHMAC:    true,
		oauthCfg:   &oauth2.Config{},
	}
}

// GoogleAuthURL returns the Google OAuth2 consent screen URL.
func (uc *AuthUseCase) GoogleAuthURL(state string) string {
	return uc.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges a Google OAuth2 code for a JWT.
func (uc *AuthUseCase) ExchangeCode(ctx context.Context, code string) (string, domain.User, error) {
	tok, err := uc.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return "", domain.User{}, fmt.Errorf("oauth exchange: %w", err)
	}

	client := uc.oauthCfg.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/userinfo/v2/me")
	if err != nil {
		return "", domain.User{}, fmt.Errorf("userinfo fetch: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var info struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", domain.User{}, fmt.Errorf("userinfo parse: %w", err)
	}

	user, err := uc.userRepo.FindOrCreate(ctx, info.Email, info.Name)
	if err != nil {
		return "", domain.User{}, fmt.Errorf("FindOrCreate: %w", err)
	}

	tokenStr, err := uc.IssueToken(user)
	if err != nil {
		return "", domain.User{}, err
	}
	return tokenStr, user, nil
}

// IssueToken signs a JWT for the given user.
func (uc *AuthUseCase) IssueToken(user domain.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	if uc.useHMAC {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		return token.SignedString(uc.hmacSecret)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(uc.privateKey)
}

// VerifyToken parses and validates a JWT string, returning the claims.
func (uc *AuthUseCase) VerifyToken(tokenStr string) (*Claims, error) {
	if uc.useHMAC {
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return uc.hmacSecret, nil
		})
		if err != nil {
			return nil, err
		}
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			return nil, fmt.Errorf("invalid token claims")
		}
		return claims, nil
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return uc.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// RefreshToken issues a new token if the existing one expires within 1 hour.
func (uc *AuthUseCase) RefreshToken(tokenStr string) (string, error) {
	claims, err := uc.VerifyToken(tokenStr)
	if err != nil {
		return "", err
	}

	exp := claims.ExpiresAt.Time
	if time.Until(exp) > time.Hour {
		return "", fmt.Errorf("token not near expiry, refresh not needed")
	}

	user := domain.User{
		ID:    claims.UserID,
		Email: claims.Email,
		Role:  domain.Role(claims.Role),
	}
	return uc.IssueToken(user)
}

// FindOrCreateUser looks up or creates a user by email.
func (uc *AuthUseCase) FindOrCreateUser(ctx context.Context, email, name string) (domain.User, error) {
	return uc.userRepo.FindOrCreate(ctx, email, name)
}

// httpClient returns the oauth HTTP client (used for testing injection).
func (uc *AuthUseCase) httpClient(ctx context.Context, tok *oauth2.Token) *http.Client {
	return uc.oauthCfg.Client(ctx, tok)
}
