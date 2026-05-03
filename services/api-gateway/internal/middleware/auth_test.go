package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app/auth"
	"apex/api-gateway/internal/domain"
	apimw "apex/api-gateway/internal/middleware"
)

// --- minimal user repo fake ---

type authTestUserRepo struct{}

func (r *authTestUserRepo) FindOrCreate(_ context.Context, email, name string) (domain.User, error) {
	return domain.User{ID: "u1", Email: email, Role: domain.RoleViewer}, nil
}
func (r *authTestUserRepo) UpdateRole(_ context.Context, _ string, _ domain.Role) error {
	return nil
}

func (r *authTestUserRepo) List(_ context.Context) ([]domain.User, error) {
	return []domain.User{}, nil
}

func newTestAuthUC() *auth.AuthUseCase {
	return auth.NewWithHMAC(&authTestUserRepo{}, []byte("test-secret"))
}

func issueToken(t *testing.T, uc *auth.AuthUseCase, user domain.User) string {
	t.Helper()
	tok, err := uc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	return tok
}

// okHandler is a downstream handler that asserts claims were set (for authenticated paths).
func okHandler(t *testing.T) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, _ := c.Get("user_id").(string)
		email, _ := c.Get("email").(string)
		role, _ := c.Get("role").(string)
		if userID == "" || email == "" || role == "" {
			t.Errorf("expected claims in context: user_id=%q email=%q role=%q", userID, email, role)
		}
		return c.String(http.StatusOK, "ok")
	}
}

// noClaimsHandler is a downstream handler for skipped paths — does NOT assert claims.
func noClaimsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}
}

func applyMiddleware(t *testing.T, uc *auth.AuthUseCase, req *http.Request) *httptest.ResponseRecorder {
	return applyMiddlewareWithHandler(t, uc, req, okHandler(t))
}

func applyMiddlewareWithHandler(t *testing.T, uc *auth.AuthUseCase, req *http.Request, downstream echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := apimw.JWTMiddleware(uc)
	handler := mw(downstream)
	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	uc := newTestAuthUC()
	user := domain.User{ID: "u1", Email: "test@test.com", Role: domain.RoleAdmin}
	tok := issueToken(t, uc, user)

	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	rec := applyMiddleware(t, uc, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	uc := newTestAuthUC()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)

	rec := applyMiddleware(t, uc, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	uc := newTestAuthUC()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")

	rec := applyMiddleware(t, uc, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	uc := newTestAuthUC()

	claims := auth.Claims{
		UserID: "u1",
		Email:  "x@y.com",
		Role:   "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	req.Header.Set("Authorization", "Bearer "+signed)

	rec := applyMiddleware(t, uc, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestJWTMiddleware_SkipsHealth(t *testing.T) {
	uc := newTestAuthUC()
	// No token — but /health should be skipped (no 401).
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := applyMiddlewareWithHandler(t, uc, req, noClaimsHandler())
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 for /health (skipped), got %d", rec.Code)
	}
}

func TestJWTMiddleware_SkipsAuthPaths(t *testing.T) {
	uc := newTestAuthUC()
	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	rec := applyMiddlewareWithHandler(t, uc, req, noClaimsHandler())
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 for /auth/google (skipped), got %d", rec.Code)
	}
}

func TestJWTMiddleware_SkipsWS(t *testing.T) {
	uc := newTestAuthUC()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := applyMiddlewareWithHandler(t, uc, req, noClaimsHandler())
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 for /ws (skipped), got %d", rec.Code)
	}
}
