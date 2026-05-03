package minio

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements app.Storage using a MinIO (S3-compatible) bucket.
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// New creates a MinIOStorage connected to the given endpoint.
// secure should be false for local MinIO instances.
func New(endpoint, accessKey, secretKey, bucket string, secure bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &MinIOStorage{client: client, bucket: bucket}, nil
}

// Upload stores data under key in the configured bucket.
func (s *MinIOStorage) Upload(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return fmt.Errorf("minio put object: %w", err)
	}
	return nil
}
