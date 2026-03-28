package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalBackend stores artifacts on the local filesystem under a base directory.
type LocalBackend struct {
	basePath string
}

// NewLocalBackend creates a LocalBackend rooted at basePath.
// The base directory is created if it does not already exist.
func NewLocalBackend(basePath string) *LocalBackend {
	return &LocalBackend{basePath: basePath}
}

func (l *LocalBackend) fullPath(key string) string {
	return filepath.Join(l.basePath, filepath.FromSlash(key))
}

// Upload writes data from reader to basePath/key, creating intermediate
// directories as needed.
func (l *LocalBackend) Upload(_ context.Context, key string, reader io.Reader, _ int64) error {
	dst := l.fullPath(key)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("local: create directory: %w", err)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("local: create file: %w", err)
	}

	if _, err := io.Copy(f, reader); err != nil {
		f.Close()
		return fmt.Errorf("local: write file: %w", err)
	}
	return f.Close()
}

// Download opens the file at basePath/key and returns a ReadCloser.
func (l *LocalBackend) Download(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(l.fullPath(key))
	if err != nil {
		return nil, fmt.Errorf("local: open file: %w", err)
	}
	return f, nil
}

// Delete removes the file at basePath/key.
func (l *LocalBackend) Delete(_ context.Context, key string) error {
	err := os.Remove(l.fullPath(key))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local: remove file: %w", err)
	}
	return nil
}

// Exists reports whether a file exists at basePath/key.
func (l *LocalBackend) Exists(_ context.Context, key string) (bool, error) {
	_, err := os.Stat(l.fullPath(key))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("local: stat file: %w", err)
}
