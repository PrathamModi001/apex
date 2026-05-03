package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"apex/api-gateway/internal/app"
	"apex/api-gateway/internal/domain"
)

// ListUsersHandler handles GET /users.
func ListUsersHandler(userRepo app.UserRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		users, err := userRepo.List(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"users": users})
	}
}

// GetAuditChainHandler handles GET /audit/:id.
func GetAuditChainHandler(auditRepo app.AuditRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		entries, err := auditRepo.GetChain(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if entries == nil {
			entries = []app.AuditEntry{}
		}
		return c.JSON(http.StatusOK, entries)
	}
}

// UpdateUserRoleHandler handles POST /admin/users/:id/role.
func UpdateUserRoleHandler(userRepo app.UserRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Param("id")

		var body struct {
			Role string `json:"role"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		role := domain.Role(body.Role)
		switch role {
		case domain.RoleAdmin, domain.RoleReviewer, domain.RoleViewer:
			// valid
		default:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role"})
		}

		if err := userRepo.UpdateRole(c.Request().Context(), userID, role); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}
}

// VerifyChainHandler handles POST /invoices/:id/verify-chain.
// Checks that each audit entry's prev_hash links to the previous entry's chain_hash.
func VerifyChainHandler(auditRepo app.AuditRepository) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		entries, err := auditRepo.GetChain(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if len(entries) == 0 {
			return c.JSON(http.StatusOK, map[string]any{"intact": true, "message": "no events"})
		}
		for i := 1; i < len(entries); i++ {
			if entries[i].PrevHash != entries[i-1].ChainHash {
				return c.JSON(http.StatusOK, map[string]any{
					"intact":  false,
					"message": fmt.Sprintf("chain break at event %d", i+1),
				})
			}
		}
		return c.JSON(http.StatusOK, map[string]any{"intact": true})
	}
}
