package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type vendorJSON struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	RiskScore       float64       `json:"risk_score"`
	InvoiceCount    int           `json:"invoice_count"`
	CorrectionCount int           `json:"correction_count"`
	BankAccounts    []interface{} `json:"bank_accounts"`
}

// ListVendorsHandler handles GET /vendors.
func ListVendorsHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := pool.Query(c.Request().Context(), `
			SELECT v.id, v.name, v.risk_score, v.correction_count, v.bank_accounts,
			       COUNT(i.id) AS invoice_count
			FROM vendors v
			LEFT JOIN invoices i ON i.vendor_id = v.id
			GROUP BY v.id
			ORDER BY v.name
		`)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		defer rows.Close()

		var vendors []vendorJSON
		for rows.Next() {
			var v vendorJSON
			var bankJSON []byte
			if err := rows.Scan(&v.ID, &v.Name, &v.RiskScore, &v.CorrectionCount, &bankJSON, &v.InvoiceCount); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if len(bankJSON) > 0 {
				_ = json.Unmarshal(bankJSON, &v.BankAccounts)
			}
			if v.BankAccounts == nil {
				v.BankAccounts = []interface{}{}
			}
			vendors = append(vendors, v)
		}
		if vendors == nil {
			vendors = []vendorJSON{}
		}
		return c.JSON(http.StatusOK, vendors)
	}
}

// GetVendorHandler handles GET /vendors/:id.
func GetVendorHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var v vendorJSON
		var bankJSON []byte
		err := pool.QueryRow(c.Request().Context(), `
			SELECT v.id, v.name, v.risk_score, v.correction_count, v.bank_accounts,
			       COUNT(i.id) AS invoice_count
			FROM vendors v
			LEFT JOIN invoices i ON i.vendor_id = v.id
			WHERE v.id = $1
			GROUP BY v.id
		`, id).Scan(&v.ID, &v.Name, &v.RiskScore, &v.CorrectionCount, &bankJSON, &v.InvoiceCount)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "vendor not found"})
		}
		if len(bankJSON) > 0 {
			_ = json.Unmarshal(bankJSON, &v.BankAccounts)
		}
		if v.BankAccounts == nil {
			v.BankAccounts = []interface{}{}
		}
		return c.JSON(http.StatusOK, v)
	}
}
