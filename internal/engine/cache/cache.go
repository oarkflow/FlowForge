package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CacheManager manages shared cache volumes across builds.
type CacheManager struct {
	cacheDir string
	maxSize  int64 // max total cache size in bytes
	mu       sync.Mutex
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(cacheDir string, maxSizeMB int64) *CacheManager {
	if cacheDir == "" {
		cacheDir = "/tmp/flowforge-cache"
	}
	os.MkdirAll(cacheDir, 0755)
	return &CacheManager{
		cacheDir: cacheDir,
		maxSize:  maxSizeMB * 1024 * 1024,
	}
}

// CacheKey generates a deterministic cache key from a file's content hash.
// This is used for lock files like go.sum, package-lock.json, etc.
func CacheKey(keyPattern, filePath string) string {
	h := sha256.New()
	h.Write([]byte(keyPattern))
	if f, err := os.Open(filePath); err == nil {
		defer f.Close()
		io.Copy(h, f)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// Restore restores a cache to the given destination directory.
// Returns true if the cache was found and restored.
func (cm *CacheManager) Restore(ctx context.Context, key string, destPaths []string) (bool, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	archivePath := filepath.Join(cm.cacheDir, key+".tar.gz")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return false, nil
	}

	// Extract to each destination path
	for _, dest := range destPaths {
		os.MkdirAll(dest, 0755)
		cmd := exec.CommandContext(ctx, "tar", "xzf", archivePath, "-C", dest)
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("cache restore failed: %w", err)
		}
	}

	// Update access time for LRU
	now := time.Now()
	os.Chtimes(archivePath, now, now)

	log.Info().Str("key", key).Msg("cache: restored")
	return true, nil
}

// Save saves the given source paths to a cache archive.
func (cm *CacheManager) Save(ctx context.Context, key string, srcPaths []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	archivePath := filepath.Join(cm.cacheDir, key+".tar.gz")

	// Build tar command args
	args := []string{"czf", archivePath}
	for _, src := range srcPaths {
		if _, err := os.Stat(src); err == nil {
			args = append(args, "-C", filepath.Dir(src), filepath.Base(src))
		}
	}

	if len(args) <= 2 {
		return nil // Nothing to cache
	}

	cmd := exec.CommandContext(ctx, "tar", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cache save failed: %w", err)
	}

	log.Info().Str("key", key).Msg("cache: saved")

	// Evict old caches if necessary
	cm.evict()

	return nil
}

// evict removes oldest cache entries when total size exceeds maxSize.
func (cm *CacheManager) evict() {
	if cm.maxSize <= 0 {
		return
	}

	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return
	}

	type cacheEntry struct {
		path    string
		size    int64
		modTime time.Time
	}

	var caches []cacheEntry
	var totalSize int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		caches = append(caches, cacheEntry{
			path:    filepath.Join(cm.cacheDir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	if totalSize <= cm.maxSize {
		return
	}

	// Sort by modification time (oldest first) for LRU eviction
	sort.Slice(caches, func(i, j int) bool {
		return caches[i].modTime.Before(caches[j].modTime)
	})

	for _, c := range caches {
		if totalSize <= cm.maxSize {
			break
		}
		os.Remove(c.path)
		totalSize -= c.size
		log.Info().Str("path", c.path).Msg("cache: evicted")
	}
}

// Clear removes all cached items.
func (cm *CacheManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		os.Remove(filepath.Join(cm.cacheDir, entry.Name()))
	}
	log.Info().Msg("cache: cleared all")
	return nil
}
