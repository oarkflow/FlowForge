package queries

import (
	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// PipelineLinkRepo handles CRUD operations for pipeline composition links.
type PipelineLinkRepo struct {
	db *sqlx.DB
}

// ListBySource returns all links where the given pipeline is the source.
func (r *PipelineLinkRepo) ListBySource(pipelineID string) ([]models.PipelineLink, error) {
	var links []models.PipelineLink
	err := r.db.Select(&links,
		"SELECT * FROM pipeline_links WHERE source_pipeline_id = ? ORDER BY created_at ASC",
		pipelineID)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// ListByTarget returns all links where the given pipeline is the target.
func (r *PipelineLinkRepo) ListByTarget(pipelineID string) ([]models.PipelineLink, error) {
	var links []models.PipelineLink
	err := r.db.Select(&links,
		"SELECT * FROM pipeline_links WHERE target_pipeline_id = ? ORDER BY created_at ASC",
		pipelineID)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// ListByPipeline returns all links where the given pipeline is either source or target.
func (r *PipelineLinkRepo) ListByPipeline(pipelineID string) ([]models.PipelineLink, error) {
	var links []models.PipelineLink
	err := r.db.Select(&links,
		"SELECT * FROM pipeline_links WHERE source_pipeline_id = ? OR target_pipeline_id = ? ORDER BY created_at ASC",
		pipelineID, pipelineID)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// GetByID returns a single pipeline link by its ID.
func (r *PipelineLinkRepo) GetByID(id string) (*models.PipelineLink, error) {
	var link models.PipelineLink
	err := r.db.Get(&link, "SELECT * FROM pipeline_links WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// Create inserts a new pipeline link.
func (r *PipelineLinkRepo) Create(link *models.PipelineLink) error {
	_, err := r.db.Exec(`
		INSERT INTO pipeline_links (source_pipeline_id, target_pipeline_id, link_type, condition, pass_variables, enabled)
		VALUES (?, ?, ?, ?, ?, ?)`,
		link.SourcePipelineID, link.TargetPipelineID, link.LinkType,
		link.Condition, link.PassVariables, link.Enabled,
	)
	if err != nil {
		return err
	}
	// Retrieve the created link to get the auto-generated ID and created_at
	return r.db.Get(link, "SELECT * FROM pipeline_links WHERE rowid = last_insert_rowid()")
}

// Update modifies an existing pipeline link.
func (r *PipelineLinkRepo) Update(link *models.PipelineLink) error {
	_, err := r.db.Exec(`
		UPDATE pipeline_links SET
			source_pipeline_id = ?, target_pipeline_id = ?, link_type = ?,
			condition = ?, pass_variables = ?, enabled = ?
		WHERE id = ?`,
		link.SourcePipelineID, link.TargetPipelineID, link.LinkType,
		link.Condition, link.PassVariables, link.Enabled, link.ID,
	)
	return err
}

// Delete removes a pipeline link by ID.
func (r *PipelineLinkRepo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM pipeline_links WHERE id = ?", id)
	return err
}
