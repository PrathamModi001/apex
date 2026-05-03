package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app/auth"
)

// JWTMiddleware returns an Echo middleware that validates Bearer tokens.
// Skips /health, /auth/*, and /ws paths.
func JWTMiddleware(authUC *auth.AuthUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip auth for public paths.
			if path == "/health" ||
				strings.HasPrefix(path, "/auth/") ||
				path == "/ws" {
				return next(c)
			}

			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := authUC.VerifyToken(tokenStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}
