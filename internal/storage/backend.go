package storage

import (
	"context"
	"io"
)

// StorageBackend defines the interface for artifact storage backends.
type StorageBackend interface {
	// Upload stores the contents of reader under the given key.
	Upload(ctx context.Context, key string, reader io.Reader, size int64) error

	// Download returns a ReadCloser for the stored object. The caller must close it.
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the stored object identified by key.
	Delete(ctx context.Context, key string) error

	// Exists checks whether an object exists for the given key.
	Exists(ctx context.Context, key string) (bool, error)
}
