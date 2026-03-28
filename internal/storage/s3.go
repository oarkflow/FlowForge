package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Backend stores artifacts in an S3-compatible object store via minio-go.
type S3Backend struct {
	client *minio.Client
	bucket string
}

// NewS3Backend creates a new S3Backend connected to the given endpoint.
// It works with AWS S3, MinIO, GCS (interop), and any other S3-compatible service.
func NewS3Backend(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*S3Backend, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: init client: %w", err)
	}
	return &S3Backend{
		client: client,
		bucket: bucket,
	}, nil
}

// ensureBucket creates the bucket if it does not already exist.
func (s *S3Backend) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("s3: check bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("s3: create bucket: %w", err)
		}
	}
	return nil
}

// Upload stores the contents of reader under the given key in the bucket.
func (s *S3Backend) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("s3: put object: %w", err)
	}
	return nil
}

// Download returns a ReadCloser for the object stored under key.
func (s *S3Backend) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: get object: %w", err)
	}
	// Verify the object is reachable by issuing a Stat.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("s3: stat object: %w", err)
	}
	return obj, nil
}

// Delete removes the object stored under key.
func (s *S3Backend) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3: remove object: %w", err)
	}
	return nil
}

// Exists checks whether an object exists under the given key.
func (s *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("s3: stat object: %w", err)
	}
	return true, nil
}
