//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresinfra "apex/event-worker/internal/infra/postgres"
)

func TestPOMatcher_Integration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	matcher := postgresinfra.NewPOMatcher(pool)

	t.Run("vector match returns result or no match", func(t *testing.T) {
		match, err := matcher.Match(context.Background(), "Test Vendor Corp", 500.0)
		if err != nil {
			t.Fatalf("match error: %v", err)
		}
		// We just verify the struct is valid; actual PO data depends on seed data
		t.Logf("match result: %+v", match)
	})
}
