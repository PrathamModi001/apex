package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type policyJSON struct {
	ID            string     `json:"id"`
	RawText       string     `json:"raw_text"`
	CompiledRule  any        `json:"compiled_rule,omitempty"`
	Active        bool       `json:"active"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
}

// ListPoliciesHandler handles GET /policies.
func ListPoliciesHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := pool.Query(c.Request().Context(),
			`SELECT id, raw_text, compiled_rule, active, last_triggered_at FROM policies ORDER BY created_at DESC`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var policies []policyJSON
		for rows.Next() {
			var p policyJSON
			var compiledJSON []byte
			var lastTriggered *time.Time
			if err := rows.Scan(&p.ID, &p.RawText, &compiledJSON, &p.Active, &lastTriggered); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if len(compiledJSON) > 0 {
				_ = json.Unmarshal(compiledJSON, &p.CompiledRule)
			}
			p.LastTriggered = lastTriggered
			policies = append(policies, p)
		}
		if policies == nil {
			policies = []policyJSON{}
		}
		return c.JSON(http.StatusOK, policies)
	}
}

// CreatePolicyHandler handles POST /policies — proxies to agent-service for LLM compilation.
func CreatePolicyHandler(agentURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var body struct {
			RawText string `json:"raw_text"`
		}
		if err := c.Bind(&body); err != nil || body.RawText == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "raw_text required"})
		}

		actor, _ := c.Get("email").(string)
		if actor == "" {
			actor, _ = c.Get("user_id").(string)
		}

		reqBody, _ := json.Marshal(map[string]string{"raw_text": body.RawText, "created_by": actor})
		resp, err := http.Post(fmt.Sprintf("%s/policies/compile", agentURL), "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]string{"error": "agent service unavailable"})
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			return c.JSON(resp.StatusCode, map[string]string{"error": string(respBody)})
		}

		var result any
		_ = json.Unmarshal(respBody, &result)
		return c.JSON(http.StatusOK, result)
	}
}

// TogglePolicyHandler handles PATCH /policies/:id.
func TogglePolicyHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var body struct {
			Active bool `json:"active"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}
		_, err := pool.Exec(c.Request().Context(),
			`UPDATE policies SET active = $1 WHERE id = $2`, body.Active, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]any{"id": id, "active": body.Active})
	}
}

// DeletePolicyHandler handles DELETE /policies/:id.
func DeletePolicyHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		tag, err := pool.Exec(c.Request().Context(), `DELETE FROM policies WHERE id = $1`, id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if tag.RowsAffected() == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "policy not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}
