package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"apex/event-worker/internal/domain"
)

// POMatcher performs vendor-to-purchase-order matching using pgvector cosine similarity.
type POMatcher struct {
	pool *pgxpool.Pool
}

// NewPOMatcher creates a POMatcher using the provided connection pool.
func NewPOMatcher(pool *pgxpool.Pool) *POMatcher {
	return &POMatcher{pool: pool}
}

// Match attempts to find the best matching purchase order for a vendor name + amount.
// It first tries an exact vendor name match, then falls back to pgvector similarity.
func (m *POMatcher) Match(ctx context.Context, vendorName string, amount float64) (domain.POMatch, error) {
	// Try exact name match first (fast path)
	match, found, err := m.exactMatch(ctx, vendorName)
	if err != nil {
		return domain.POMatch{}, err
	}
	if found {
		return match, nil
	}

	// Fallback: pgvector cosine similarity on vendor embeddings
	return m.vectorMatch(ctx, vendorName)
}

// exactMatch performs a case-insensitive exact vendor name lookup.
func (m *POMatcher) exactMatch(ctx context.Context, vendorName string) (domain.POMatch, bool, error) {
	const q = `
		SELECT po.id
		FROM vendors v
		JOIN purchase_orders po ON po.vendor_id = v.id
		WHERE LOWER(v.name) = LOWER($1)
		LIMIT 1`

	var poID string
	err := m.pool.QueryRow(ctx, q, vendorName).Scan(&poID)
	if err != nil {
		// pgx returns pgx.ErrNoRows when no result
		return domain.POMatch{}, false, nil
	}
	return domain.POMatch{
		POID:       poID,
		Confidence: 1.0,
		Matched:    true,
	}, true, nil
}

// vectorMatch uses pgvector cosine distance to find the closest vendor embedding.
// The embedding for the query is a simple hash-based placeholder; in production,
// replace the hash with a call to Groq's embeddings API.
func (m *POMatcher) vectorMatch(ctx context.Context, vendorName string) (domain.POMatch, error) {
	// Build a deterministic float32 slice from the vendor name (hash-based stub).
	// Production code should call Groq /openai/v1/embeddings here.
	embedding := hashEmbedding(vendorName, 1536)

	const q = `
		SELECT po.id, 1 - (v.embedding <=> $1::vector) AS confidence
		FROM vendors v
		JOIN purchase_orders po ON po.vendor_id = v.id
		WHERE v.embedding IS NOT NULL
		ORDER BY confidence DESC
		LIMIT 1`

	var poID string
	var confidence float64
	err := m.pool.QueryRow(ctx, q, pgvectorLiteral(embedding)).Scan(&poID, &confidence)
	if err != nil {
		// No matching PO found — return unmatched result (not an error)
		return domain.POMatch{Matched: false, Confidence: 0}, nil
	}

	matched := confidence >= 0.7
	return domain.POMatch{
		POID:       poID,
		Confidence: confidence,
		Matched:    matched,
	}, nil
}

// hashEmbedding produces a deterministic float32 slice from a string.
// This is a placeholder; replace with a real embedding API call in production.
func hashEmbedding(s string, dims int) []float32 {
	vec := make([]float32, dims)
	for i, ch := range s {
		vec[i%dims] += float32(ch) / 255.0
	}
	// L2 normalise
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		sqrtNorm := float32(1.0)
		// Simple approximation: divide each element by the max to keep in [0,1]
		var maxVal float32
		for _, v := range vec {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal > 0 {
			sqrtNorm = maxVal
		}
		for i := range vec {
			vec[i] /= sqrtNorm
		}
	}
	return vec
}

// pgvectorLiteral converts a []float32 to a Postgres vector literal string.
func pgvectorLiteral(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	s := "["
	for i, v := range vec {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%f", v)
	}
	s += "]"
	return s
}
