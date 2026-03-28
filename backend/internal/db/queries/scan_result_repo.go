package queries

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/models"
)

type ScanResultRepo struct {
	db *sqlx.DB
}

func (r *ScanResultRepo) GetByID(ctx context.Context, id string) (*models.ScanResult, error) {
	s := &models.ScanResult{}
	err := r.db.GetContext(ctx, s, "SELECT * FROM scan_results WHERE id = ?", id)
	return s, err
}

func (r *ScanResultRepo) ListByRunID(ctx context.Context, runID string, limit, offset int) ([]models.ScanResult, error) {
	results := []models.ScanResult{}
	err := r.db.SelectContext(ctx, &results, "SELECT * FROM scan_results WHERE run_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", runID, limit, offset)
	return results, err
}

func (r *ScanResultRepo) Create(ctx context.Context, s *models.ScanResult) error {
	s.ID = uuid.New().String()
	s.CreatedAt = time.Now()
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO scan_results (id, run_id, scanner_type, target, vulnerabilities, critical_count, high_count, medium_count, low_count, status, created_at)
		VALUES (:id, :run_id, :scanner_type, :target, :vulnerabilities, :critical_count, :high_count, :medium_count, :low_count, :status, :created_at)`,
		s)
	return err
}

func (r *ScanResultRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scan_results WHERE id = ?", id)
	return err
}
