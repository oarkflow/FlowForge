package admin

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// BackupInfo represents metadata about a backup.
type BackupInfo struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
	Path      string    `json:"path"`
}

// BackupService manages SQLite database backups.
type BackupService struct {
	db        *sqlx.DB
	backupDir string
}

// NewBackupService creates a new backup service.
func NewBackupService(db *sqlx.DB, backupDir string) *BackupService {
	if backupDir == "" {
		backupDir = "data/backups"
	}
	os.MkdirAll(backupDir, 0755)
	return &BackupService{
		db:        db,
		backupDir: backupDir,
	}
}

// CreateBackup creates a compressed backup of the SQLite database using VACUUM INTO.
func (s *BackupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("flowforge-backup-%s.db", timestamp)
	dbPath := filepath.Join(s.backupDir, filename)
	gzPath := dbPath + ".gz"

	log.Info().Str("path", gzPath).Msg("backup: creating backup")

	// Use VACUUM INTO for a consistent snapshot
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", dbPath))
	if err != nil {
		return nil, fmt.Errorf("backup: VACUUM INTO failed: %w", err)
	}

	// Compress the backup
	if err := compressFile(dbPath, gzPath); err != nil {
		os.Remove(dbPath)
		return nil, fmt.Errorf("backup: compression failed: %w", err)
	}

	// Remove uncompressed file
	os.Remove(dbPath)

	// Get file info
	info, err := os.Stat(gzPath)
	if err != nil {
		return nil, fmt.Errorf("backup: stat failed: %w", err)
	}

	backup := &BackupInfo{
		ID:        timestamp,
		Filename:  filename + ".gz",
		SizeBytes: info.Size(),
		CreatedAt: time.Now(),
		Path:      gzPath,
	}

	log.Info().
		Str("filename", backup.Filename).
		Int64("size", backup.SizeBytes).
		Msg("backup: created successfully")

	return backup, nil
}

// ListBackups returns all available backups sorted by creation time (newest first).
func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("backup: list failed: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			ID:        entry.Name(),
			Filename:  entry.Name(),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime(),
			Path:      filepath.Join(s.backupDir, entry.Name()),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// RestoreBackup restores the database from a backup file.
// WARNING: This replaces the current database.
func (s *BackupService) RestoreBackup(ctx context.Context, backupPath string) error {
	log.Warn().Str("path", backupPath).Msg("backup: restoring database")

	// Verify the backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup: file not found: %w", err)
	}

	// Get current database path from the connection
	var dbPath string
	row := s.db.QueryRowContext(ctx, "PRAGMA database_list")
	var seq int
	var name string
	if err := row.Scan(&seq, &name, &dbPath); err != nil {
		return fmt.Errorf("backup: cannot determine database path: %w", err)
	}

	// Decompress if gzipped
	restorePath := backupPath
	if filepath.Ext(backupPath) == ".gz" {
		tmpPath := filepath.Join(s.backupDir, "restore-temp.db")
		if err := decompressFile(backupPath, tmpPath); err != nil {
			return fmt.Errorf("backup: decompression failed: %w", err)
		}
		restorePath = tmpPath
		defer os.Remove(tmpPath)
	}

	// Close the database connection
	s.db.Close()

	// Copy the restore file to the database path
	src, err := os.Open(restorePath)
	if err != nil {
		return fmt.Errorf("backup: open restore file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("backup: create database: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("backup: copy failed: %w", err)
	}

	log.Info().Msg("backup: restored successfully — server restart required")
	return nil
}

// ScheduledBackup runs a backup on a periodic interval.
func (s *BackupService) ScheduledBackup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.CreateBackup(ctx); err != nil {
				log.Error().Err(err).Msg("backup: scheduled backup failed")
			}
		}
	}
}

func compressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	w := gzip.NewWriter(out)
	defer w.Close()

	_, err = io.Copy(w, in)
	return err
}

func decompressFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	r, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer r.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	return err
}
