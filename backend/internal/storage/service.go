package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// ArtifactService coordinates artifact storage and database metadata.
type ArtifactService struct {
	backend StorageBackend
	repos   *queries.Repositories
}

// NewArtifactService creates a new ArtifactService.
func NewArtifactService(backend StorageBackend, repos *queries.Repositories) *ArtifactService {
	return &ArtifactService{
		backend: backend,
		repos:   repos,
	}
}

// Upload reads data from reader, computes its SHA-256 checksum, stores the file
// via the configured storage backend, and creates a corresponding database record.
func (s *ArtifactService) Upload(ctx context.Context, runID, stepRunID, name string, reader io.Reader, size int64) (*models.Artifact, error) {
	key := fmt.Sprintf("%s/%s/%s", runID, uuid.New().String(), name)

	// Wrap reader in a hash writer so we compute the checksum on the fly.
	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := s.backend.Upload(ctx, key, tee, size); err != nil {
		return nil, fmt.Errorf("artifact upload: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	sizeInt := int(size)

	a := &models.Artifact{
		RunID:          runID,
		Name:           name,
		Path:           name,
		SizeBytes:      &sizeInt,
		ChecksumSHA256: &checksum,
		StorageBackend: backendName(s.backend),
		StorageKey:     key,
	}

	if stepRunID != "" {
		a.StepRunID = &stepRunID
	}

	if err := s.repos.Artifacts.Create(ctx, a); err != nil {
		// Best-effort cleanup of the already-uploaded file.
		_ = s.backend.Delete(ctx, key)
		return nil, fmt.Errorf("artifact db create: %w", err)
	}

	return a, nil
}

// Download retrieves an artifact's metadata from the database and returns a
// reader for its contents from the storage backend.
func (s *ArtifactService) Download(ctx context.Context, artifactID string) (io.ReadCloser, *models.Artifact, error) {
	a, err := s.repos.Artifacts.GetByID(ctx, artifactID)
	if err != nil {
		return nil, nil, fmt.Errorf("artifact db get: %w", err)
	}

	rc, err := s.backend.Download(ctx, a.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("artifact download: %w", err)
	}

	return rc, a, nil
}

// List returns all artifacts for a given pipeline run.
func (s *ArtifactService) List(ctx context.Context, runID string) ([]models.Artifact, error) {
	return s.repos.Artifacts.ListByRunID(ctx, runID)
}

// Delete removes an artifact from both the storage backend and the database.
func (s *ArtifactService) Delete(ctx context.Context, artifactID string) error {
	a, err := s.repos.Artifacts.GetByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("artifact db get: %w", err)
	}

	if err := s.backend.Delete(ctx, a.StorageKey); err != nil {
		return fmt.Errorf("artifact storage delete: %w", err)
	}

	if err := s.repos.Artifacts.Delete(ctx, artifactID); err != nil {
		return fmt.Errorf("artifact db delete: %w", err)
	}

	return nil
}

// CleanupExpired deletes all artifacts whose expiry date has passed.
// It fetches expired artifacts, removes their backing storage objects, then
// deletes the database rows. Returns the number of artifacts removed.
func (s *ArtifactService) CleanupExpired(ctx context.Context) (int, error) {
	expired, err := s.repos.Artifacts.ListExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("artifact list expired: %w", err)
	}

	var count int
	for _, a := range expired {
		// Best-effort removal from storage; continue even if it fails.
		_ = s.backend.Delete(ctx, a.StorageKey)
		if err := s.repos.Artifacts.Delete(ctx, a.ID); err == nil {
			count++
		}
	}

	return count, nil
}

// backendName returns a human-readable name for a storage backend.
func backendName(b StorageBackend) string {
	switch b.(type) {
	case *LocalBackend:
		return "local"
	case *S3Backend:
		return "s3"
	default:
		return "unknown"
	}
}
