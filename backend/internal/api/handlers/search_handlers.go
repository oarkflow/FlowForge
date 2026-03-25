package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
)

// searchResult holds categorized search results.
type searchResult struct {
	Projects  []models.Project     `json:"projects"`
	Pipelines []models.Pipeline    `json:"pipelines"`
	Runs      []models.PipelineRun `json:"runs"`
}

// GlobalSearch searches across projects, pipelines, and pipeline runs.
// GET /api/v1/search?q=<query>
func (h *Handler) GlobalSearch(c fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.JSON(searchResult{
			Projects:  []models.Project{},
			Pipelines: []models.Pipeline{},
			Runs:      []models.PipelineRun{},
		})
	}

	pattern := "%" + q + "%"
	const limit = 5

	// Search projects by name
	var projects []models.Project
	err := h.db.SelectContext(c.Context(), &projects,
		"SELECT * FROM projects WHERE deleted_at IS NULL AND name LIKE ? ORDER BY created_at DESC LIMIT ?",
		pattern, limit)
	if err != nil {
		projects = []models.Project{}
	}

	// Search pipelines by name
	var pipelines []models.Pipeline
	err = h.db.SelectContext(c.Context(), &pipelines,
		"SELECT * FROM pipelines WHERE deleted_at IS NULL AND name LIKE ? ORDER BY created_at DESC LIMIT ?",
		pattern, limit)
	if err != nil {
		pipelines = []models.Pipeline{}
	}

	// Search pipeline runs by commit SHA or branch
	var runs []models.PipelineRun
	err = h.db.SelectContext(c.Context(), &runs,
		"SELECT * FROM pipeline_runs WHERE (commit_sha LIKE ? OR branch LIKE ?) ORDER BY created_at DESC LIMIT ?",
		pattern, pattern, limit)
	if err != nil {
		runs = []models.PipelineRun{}
	}

	return c.JSON(searchResult{
		Projects:  projects,
		Pipelines: pipelines,
		Runs:      runs,
	})
}
