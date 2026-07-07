// Package miniostore implements the object-storage contract on MinIO.
package miniostore

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
)

type Store struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func New(client *minio.Client, bucket, publicBaseURL string) *Store {
	return &Store{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimSuffix(publicBaseURL, "/"),
	}
}

func (s *Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("failed to store object %s: %w", key, err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

func (s *Store) PublicURL(key string) string {
	if key == "" {
		return ""
	}
	return s.publicBaseURL + "/" + key
}
