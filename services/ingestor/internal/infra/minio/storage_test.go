package minio_test

import (
	"testing"

	minioinfra "apex/ingestor/internal/infra/minio"
)

// TestNew_InvalidEndpoint verifies the constructor handles unreachable endpoints
// gracefully (minio-go performs lazy connection, so New itself should succeed).
func TestNew_InvalidEndpoint(t *testing.T) {
	_, err := minioinfra.New("localhost:19999", "user", "pass", "bucket", false)
	// minio-go does lazy dialing — constructor should not return an error
	if err != nil {
		t.Fatalf("expected no error constructing client, got: %v", err)
	}
}
