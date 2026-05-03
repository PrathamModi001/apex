package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/domain"
	apimw "apex/api-gateway/internal/middleware"
)

func applyRBACMiddleware(role string, allowed ...domain.Role) *httptest.ResponseRecorder {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if role != "" {
		c.Set("role", role)
	}

	mw := apimw.RequireRole(allowed...)
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}

func TestRBAC_AllowsCorrectRole(t *testing.T) {
	rec := applyRBACMiddleware(string(domain.RoleAdmin), domain.RoleAdmin, domain.RoleReviewer)
	if rec.Code != http.StatusOK {
		t.Errorf("admin should be allowed, got %d", rec.Code)
	}
}

func TestRBAC_AllowsReviewer(t *testing.T) {
	rec := applyRBACMiddleware(string(domain.RoleReviewer), domain.RoleAdmin, domain.RoleReviewer)
	if rec.Code != http.StatusOK {
		t.Errorf("reviewer should be allowed, got %d", rec.Code)
	}
}

func TestRBAC_DeniesViewer(t *testing.T) {
	rec := applyRBACMiddleware(string(domain.RoleViewer), domain.RoleAdmin, domain.RoleReviewer)
	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer should be forbidden, got %d", rec.Code)
	}
}

func TestRBAC_DeniesEmptyRole(t *testing.T) {
	rec := applyRBACMiddleware("", domain.RoleAdmin)
	if rec.Code != http.StatusForbidden {
		t.Errorf("empty role should be forbidden, got %d", rec.Code)
	}
}

func TestRBAC_AdminOnly(t *testing.T) {
	cases := []struct {
		role    string
		wantOK  bool
	}{
		{string(domain.RoleAdmin), true},
		{string(domain.RoleReviewer), false},
		{string(domain.RoleViewer), false},
		{"", false},
	}
	for _, tc := range cases {
		rec := applyRBACMiddleware(tc.role, domain.RoleAdmin)
		if tc.wantOK && rec.Code != http.StatusOK {
			t.Errorf("role=%q: want 200, got %d", tc.role, rec.Code)
		}
		if !tc.wantOK && rec.Code != http.StatusForbidden {
			t.Errorf("role=%q: want 403, got %d", tc.role, rec.Code)
		}
	}
}

func TestRBAC_ViewerAllowed(t *testing.T) {
	rec := applyRBACMiddleware(string(domain.RoleViewer),
		domain.RoleAdmin, domain.RoleReviewer, domain.RoleViewer)
	if rec.Code != http.StatusOK {
		t.Errorf("viewer should be allowed when in list, got %d", rec.Code)
	}
}
