package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Reader downloads files from MinIO object storage.
type Reader struct {
	client *miniogo.Client
	bucket string
}

// NewReader creates a MinIO reader connecting to the given endpoint.
func NewReader(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*Reader, error) {
	client, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}
	return &Reader{client: client, bucket: bucket}, nil
}

// Download retrieves an object by key and returns its bytes and content type.
func (r *Reader) Download(ctx context.Context, key string) ([]byte, string, error) {
	obj, err := r.client.GetObject(ctx, r.bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("get object %q: %w", key, err)
	}
	defer obj.Close()

	info, statErr := obj.Stat()
	if statErr != nil {
		return nil, "", fmt.Errorf("stat object %q: %w", key, statErr)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, obj); err != nil {
		return nil, "", fmt.Errorf("read object %q: %w", key, err)
	}

	return buf.Bytes(), info.ContentType, nil
}
