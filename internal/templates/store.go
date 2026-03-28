package templates

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// Store manages pipeline templates, including built-in and community templates.
type Store struct {
	repo *queries.TemplateRepo
}

// NewStore creates a new template store.
func NewStore(repo *queries.TemplateRepo) *Store {
	return &Store{repo: repo}
}

// SeedBuiltins inserts or updates built-in templates in the database.
// Called at startup to ensure all built-in templates are available.
func (s *Store) SeedBuiltins(ctx context.Context) error {
	for _, t := range BuiltinTemplates {
		existing, err := s.repo.GetByID(ctx, t.ID)
		if err != nil {
			// Does not exist, create it
			if err := s.repo.Create(ctx, &t); err != nil {
				log.Warn().Err(err).Str("template", t.Name).Msg("templates: failed to seed builtin")
			}
			continue
		}
		// Update the config if it changed
		existing.Config = t.Config
		existing.Description = t.Description
		existing.Category = t.Category
		_ = s.repo.Update(ctx, existing)
	}
	return nil
}

// List returns templates filtered by category and builtin-only flag.
func (s *Store) List(ctx context.Context, category string, builtinOnly bool, limit, offset int) ([]models.PipelineTemplate, error) {
	return s.repo.List(ctx, category, builtinOnly, limit, offset)
}

// Get returns a single template by ID and increments its download counter.
func (s *Store) Get(ctx context.Context, id string) (*models.PipelineTemplate, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	// Increment downloads
	_ = s.repo.IncrementDownloads(ctx, id)
	return t, nil
}

// Create creates a new community template.
func (s *Store) Create(ctx context.Context, t *models.PipelineTemplate) error {
	t.IsBuiltin = 0
	return s.repo.Create(ctx, t)
}

// Update updates an existing template. Built-in templates cannot be modified by users.
func (s *Store) Update(ctx context.Context, t *models.PipelineTemplate) error {
	existing, err := s.repo.GetByID(ctx, t.ID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}
	if existing.IsBuiltin == 1 {
		return fmt.Errorf("cannot modify built-in templates")
	}
	return s.repo.Update(ctx, t)
}

// Delete removes a template. Built-in templates cannot be deleted by users.
func (s *Store) Delete(ctx context.Context, id string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}
	if existing.IsBuiltin == 1 {
		return fmt.Errorf("cannot delete built-in templates")
	}
	return s.repo.Delete(ctx, id)
}
