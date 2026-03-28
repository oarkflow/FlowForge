package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/pipeline"
)

// =========================================================================
// PIPELINE LINK HANDLERS (Composition)
// =========================================================================

// ListPipelineLinks returns all composition links for a pipeline (source + target).
// GET /api/v1/pipelines/:id/links
func (h *Handler) ListPipelineLinks(c fiber.Ctx) error {
	pipelineID := c.Params("id")

	links, err := h.repo.PipelineLinks.ListByPipeline(pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list pipeline links")
	}
	return c.JSON(links)
}

type createPipelineLinkInput struct {
	TargetPipelineID string `json:"target_pipeline_id"`
	LinkType         string `json:"link_type"`
	Condition        string `json:"condition"`
	PassVariables    bool   `json:"pass_variables"`
	Enabled          *bool  `json:"enabled"`
}

// CreatePipelineLink creates a new composition link from the given pipeline to a target.
// POST /api/v1/pipelines/:id/links
func (h *Handler) CreatePipelineLink(c fiber.Ctx) error {
	sourcePipelineID := c.Params("id")

	var input createPipelineLinkInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.TargetPipelineID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "target_pipeline_id is required")
	}
	if sourcePipelineID == input.TargetPipelineID {
		return fiber.NewError(fiber.StatusBadRequest, "source and target pipeline cannot be the same")
	}

	// Validate link type
	linkType := input.LinkType
	if linkType == "" {
		linkType = "trigger"
	}
	if linkType != "trigger" && linkType != "fan_out" && linkType != "fan_in" {
		return fiber.NewError(fiber.StatusBadRequest, "link_type must be trigger, fan_out, or fan_in")
	}

	// Validate condition
	condition := input.Condition
	if condition == "" {
		condition = "success"
	}
	if condition != "success" && condition != "failure" && condition != "always" {
		return fiber.NewError(fiber.StatusBadRequest, "condition must be success, failure, or always")
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	link := &models.PipelineLink{
		SourcePipelineID: sourcePipelineID,
		TargetPipelineID: input.TargetPipelineID,
		LinkType:         linkType,
		Condition:        condition,
		PassVariables:    input.PassVariables,
		Enabled:          enabled,
	}

	if err := h.repo.PipelineLinks.Create(link); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create pipeline link: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "pipeline_link", link.ID,
		fiber.Map{"source": sourcePipelineID, "target": input.TargetPipelineID, "link_type": linkType})

	return c.Status(fiber.StatusCreated).JSON(link)
}

type updatePipelineLinkInput struct {
	LinkType      *string `json:"link_type"`
	Condition     *string `json:"condition"`
	PassVariables *bool   `json:"pass_variables"`
	Enabled       *bool   `json:"enabled"`
}

// UpdatePipelineLink updates an existing pipeline composition link.
// PUT /api/v1/pipeline-links/:lid
func (h *Handler) UpdatePipelineLink(c fiber.Ctx) error {
	linkID := c.Params("lid")

	existing, err := h.repo.PipelineLinks.GetByID(linkID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline link not found")
	}

	var input updatePipelineLinkInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.LinkType != nil && *input.LinkType != "" {
		lt := *input.LinkType
		if lt != "trigger" && lt != "fan_out" && lt != "fan_in" {
			return fiber.NewError(fiber.StatusBadRequest, "link_type must be trigger, fan_out, or fan_in")
		}
		existing.LinkType = lt
	}
	if input.Condition != nil && *input.Condition != "" {
		cond := *input.Condition
		if cond != "success" && cond != "failure" && cond != "always" {
			return fiber.NewError(fiber.StatusBadRequest, "condition must be success, failure, or always")
		}
		existing.Condition = cond
	}
	if input.PassVariables != nil {
		existing.PassVariables = *input.PassVariables
	}
	if input.Enabled != nil {
		existing.Enabled = *input.Enabled
	}

	if err := h.repo.PipelineLinks.Update(existing); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update pipeline link")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "pipeline_link", linkID, input)

	return c.JSON(existing)
}

// DeletePipelineLink deletes a pipeline composition link.
// DELETE /api/v1/pipeline-links/:lid
func (h *Handler) DeletePipelineLink(c fiber.Ctx) error {
	linkID := c.Params("lid")

	if _, err := h.repo.PipelineLinks.GetByID(linkID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline link not found")
	}

	if err := h.repo.PipelineLinks.Delete(linkID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete pipeline link")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "pipeline_link", linkID, nil)

	return c.JSON(fiber.Map{"message": "pipeline link deleted"})
}

// GetPipelineDAG computes and returns the DAG (directed acyclic graph) for a pipeline.
// GET /api/v1/pipelines/:id/dag
func (h *Handler) GetPipelineDAG(c fiber.Ctx) error {
	pipelineID := c.Params("id")

	// Fetch the pipeline
	p, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}

	// If there's no config content, return an empty DAG
	if p.ConfigContent == nil || *p.ConfigContent == "" {
		return c.JSON(pipeline.DAG{
			Nodes:    map[string]*pipeline.DAGNode{},
			Levels:   [][]string{},
			HasCycle: false,
		})
	}

	// Parse the pipeline spec
	spec, err := pipeline.Parse(*p.ConfigContent)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to parse pipeline config: "+err.Error())
	}

	// Determine stage ordering
	stageNames := spec.Stages
	if len(stageNames) == 0 {
		seen := make(map[string]bool)
		for _, job := range spec.Jobs {
			stageName := job.Stage
			if stageName == "" {
				stageName = "default"
			}
			if !seen[stageName] {
				stageNames = append(stageNames, stageName)
				seen[stageName] = true
			}
		}
	}

	// Get the dependency map
	depsMap := spec.StageNeeds
	if depsMap == nil {
		depsMap = make(map[string][]string)
	}

	// Build the DAG
	dag, err := pipeline.BuildStageDAG(stageNames, depsMap)
	if err != nil {
		// Return the DAG even if there's a cycle - the DAG struct has HasCycle flag
		return c.JSON(dag)
	}

	return c.JSON(dag)
}
