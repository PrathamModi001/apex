package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/domain"
)

// RequireRole returns an Echo middleware that enforces role-based access control.
// The request must have a "role" value in the echo context (set by JWTMiddleware).
func RequireRole(roles ...domain.Role) echo.MiddlewareFunc {
	allowed := make(map[domain.Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			roleVal, _ := c.Get("role").(string)
			role := domain.Role(roleVal)

			if _, ok := allowed[role]; !ok {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
			}

			return next(c)
		}
	}
}
