package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBackend_UploadDownload(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	content := "hello world artifact data"
	err := backend.Upload(ctx, "artifacts/test.txt", bytes.NewReader([]byte(content)), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}

	// Verify file exists on disk
	fullPath := filepath.Join(dir, "artifacts", "test.txt")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("file should exist on disk")
	}

	// Download and verify
	reader, err := backend.Download(ctx, "artifacts/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("downloaded content = %q, want %q", string(data), content)
	}
}

func TestLocalBackend_Upload_CreatesIntermediateDirs(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	err := backend.Upload(ctx, "deep/nested/path/file.bin", bytes.NewReader([]byte("data")), 4)
	if err != nil {
		t.Fatal(err)
	}

	fullPath := filepath.Join(dir, "deep", "nested", "path", "file.bin")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("deeply nested file should exist")
	}
}

func TestLocalBackend_Download_NotFound(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	_, err := backend.Download(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("should return error for nonexistent file")
	}
}

func TestLocalBackend_Delete(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	backend.Upload(ctx, "to-delete.txt", bytes.NewReader([]byte("temp")), 4)

	err := backend.Delete(ctx, "to-delete.txt")
	if err != nil {
		t.Fatal(err)
	}

	exists, err := backend.Exists(ctx, "to-delete.txt")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("file should not exist after delete")
	}
}

func TestLocalBackend_Delete_Nonexistent(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	// Deleting a nonexistent file should not error
	err := backend.Delete(ctx, "does-not-exist.txt")
	if err != nil {
		t.Errorf("deleting nonexistent file should not error: %v", err)
	}
}

func TestLocalBackend_Exists_True(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	backend.Upload(ctx, "exists.txt", bytes.NewReader([]byte("data")), 4)

	exists, err := backend.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("file should exist")
	}
}

func TestLocalBackend_Exists_False(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	exists, err := backend.Exists(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("nonexistent file should not exist")
	}
}

func TestLocalBackend_Upload_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	err := backend.Upload(ctx, "empty.txt", bytes.NewReader(nil), 0)
	if err != nil {
		t.Fatal(err)
	}

	exists, _ := backend.Exists(ctx, "empty.txt")
	if !exists {
		t.Error("empty file should still exist")
	}
}

func TestLocalBackend_Upload_LargeContent(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	// 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := backend.Upload(ctx, "large.bin", bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	reader, err := backend.Download(ctx, "large.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(downloaded) != len(data) {
		t.Errorf("downloaded size = %d, want %d", len(downloaded), len(data))
	}
}
