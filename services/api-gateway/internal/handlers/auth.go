package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app/auth"
)

// AuthHandler groups HTTP handlers for authentication endpoints.
type AuthHandler struct {
	uc *auth.AuthUseCase
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(uc *auth.AuthUseCase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// GoogleLogin handles GET /auth/google — redirects to Google OAuth consent.
func (h *AuthHandler) GoogleLogin(c echo.Context) error {
	state := "apex-state" // In production, use a CSRF-safe random state.
	url := h.uc.GoogleAuthURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles GET /auth/google/callback — exchanges code, issues JWT.
func (h *AuthHandler) GoogleCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing code"})
	}

	tokenStr, user, err := h.uc.ExchangeCode(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Set JWT as an HttpOnly cookie.
	cookie := new(http.Cookie)
	cookie.Name = "apex_token"
	cookie.Value = tokenStr
	cookie.HttpOnly = true
	cookie.Path = "/"
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": tokenStr,
		"user": map[string]string{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  string(user.Role),
		},
	})
}
