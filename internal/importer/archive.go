package importer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxArchiveSize = 500 * 1024 * 1024 // 500 MB
	maxFileCount   = 100_000
)

// Extract extracts a .zip, .tar.gz, or .tgz archive into destDir.
// It enforces size and file count limits and rejects zip-slip paths.
func Extract(archivePath, destDir string) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}
	if info.Size() > maxArchiveSize {
		return fmt.Errorf("archive exceeds maximum size of %d bytes", maxArchiveSize)
	}

	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, destDir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive format: %s (supported: .zip, .tar.gz, .tgz)", filepath.Ext(archivePath))
	}
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		count++
		if count > maxFileCount {
			return fmt.Errorf("archive exceeds maximum file count of %d", maxFileCount)
		}

		target, err := sanitizePath(destDir, f.Name)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}

		outFile, err := os.Create(target)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file: %w", err)
		}

		_, err = io.Copy(outFile, io.LimitReader(rc, maxArchiveSize))
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}
	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	count := 0

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		count++
		if count > maxFileCount {
			return fmt.Errorf("archive exceeds maximum file count of %d", maxFileCount)
		}

		target, err := sanitizePath(destDir, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			_, err = io.Copy(outFile, io.LimitReader(tr, maxArchiveSize))
			outFile.Close()
			if err != nil {
				return fmt.Errorf("write file: %w", err)
			}
		default:
			// Skip symlinks, hard links, etc.
			continue
		}
	}
	return nil
}

// sanitizePath prevents zip-slip attacks by ensuring the target path stays within destDir.
func sanitizePath(destDir, name string) (string, error) {
	// Clean the name to resolve any ".." components.
	clean := filepath.Clean(name)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid path in archive: %s (potential zip-slip attack)", name)
	}

	target := filepath.Join(destDir, clean)
	// Ensure the resolved path is within destDir.
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path in archive: %s (escapes destination directory)", name)
	}
	return target, nil
}

// UnwrapSingleSubfolder checks whether dir contains exactly one entry and
// that entry is a directory. If so it returns the path to that inner directory.
// This handles the common case where a zip archive wraps everything inside a
// single top-level folder (e.g. "my-project/") so the detector sees the
// project root directly instead of an empty-looking wrapper.
func UnwrapSingleSubfolder(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}

	// Filter out hidden files/dirs (like __MACOSX) that archives sometimes include.
	var visible []os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") && e.Name() != "__MACOSX" {
			visible = append(visible, e)
		}
	}

	if len(visible) == 1 && visible[0].IsDir() {
		return filepath.Join(dir, visible[0].Name())
	}
	return dir
}
