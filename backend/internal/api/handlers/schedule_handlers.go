package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/scheduler"
)

// scheduleService returns a scheduler.Service from the handler's repos.
// This is re-created on each call since it's a lightweight wrapper around repos.
func (h *Handler) scheduleService() *scheduler.Service {
	return scheduler.NewService(h.repo)
}

// --------------------------------------------------------------------------
// Pipeline Schedules
// --------------------------------------------------------------------------

// ListPipelineSchedules returns all schedules for a given pipeline.
// GET /api/v1/pipelines/:id/schedules
func (h *Handler) ListPipelineSchedules(c fiber.Ctx) error {
	pipelineID := c.Params("id")

	schedules, err := h.repo.Schedules.ListByPipeline(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list schedules")
	}
	return c.JSON(schedules)
}

// ListProjectSchedules returns all schedules for a given project.
// GET /api/v1/projects/:id/schedules
func (h *Handler) ListProjectSchedules(c fiber.Ctx) error {
	projectID := c.Params("id")

	schedules, err := h.repo.Schedules.ListByProject(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list schedules")
	}
	return c.JSON(schedules)
}

type createScheduleInput struct {
	CronExpression string            `json:"cron_expression"`
	Timezone       string            `json:"timezone"`
	Description    string            `json:"description"`
	Branch         string            `json:"branch"`
	EnvironmentID  *string           `json:"environment_id"`
	Variables      map[string]string `json:"variables"`
}

// CreateSchedule creates a new schedule for a pipeline.
// POST /api/v1/pipelines/:id/schedules
func (h *Handler) CreateSchedule(c fiber.Ctx) error {
	pipelineID := c.Params("id")

	var input createScheduleInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.CronExpression == "" {
		return fiber.NewError(fiber.StatusBadRequest, "cron_expression is required")
	}

	// Validate the cron expression
	if err := scheduler.ValidateCron(input.CronExpression); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cron expression: "+err.Error())
	}

	// Verify pipeline exists and get project_id
	p, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}

	tz := input.Timezone
	if tz == "" {
		tz = "UTC"
	}

	// Validate timezone
	if _, err := time.LoadLocation(tz); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid timezone: "+tz)
	}

	branch := input.Branch
	if branch == "" {
		branch = "main"
	}

	varsJSON := "{}"
	if input.Variables != nil && len(input.Variables) > 0 {
		b, _ := json.Marshal(input.Variables)
		varsJSON = string(b)
	}

	schedule := &models.PipelineSchedule{
		PipelineID:     pipelineID,
		ProjectID:      p.ProjectID,
		CronExpression: input.CronExpression,
		Timezone:       tz,
		Description:    input.Description,
		Enabled:        true,
		Branch:         branch,
		EnvironmentID:  input.EnvironmentID,
		Variables:      varsJSON,
		CreatedBy:      getUserID(c),
	}

	svc := h.scheduleService()
	if err := svc.Create(c.Context(), schedule); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create schedule: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "pipeline_schedule", schedule.ID,
		fiber.Map{"pipeline_id": pipelineID, "cron": input.CronExpression})

	return c.Status(fiber.StatusCreated).JSON(schedule)
}

type updateScheduleInput struct {
	CronExpression *string           `json:"cron_expression"`
	Timezone       *string           `json:"timezone"`
	Description    *string           `json:"description"`
	Branch         *string           `json:"branch"`
	EnvironmentID  *string           `json:"environment_id"`
	Variables      map[string]string `json:"variables"`
	Enabled        *bool             `json:"enabled"`
}

// UpdateSchedule updates an existing schedule.
// PUT /api/v1/schedules/:sid
func (h *Handler) UpdateSchedule(c fiber.Ctx) error {
	scheduleID := c.Params("sid")

	existing, err := h.repo.Schedules.GetByID(c.Context(), scheduleID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}

	var input updateScheduleInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.CronExpression != nil && *input.CronExpression != "" {
		if err := scheduler.ValidateCron(*input.CronExpression); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid cron expression: "+err.Error())
		}
		existing.CronExpression = *input.CronExpression
	}

	if input.Timezone != nil && *input.Timezone != "" {
		if _, err := time.LoadLocation(*input.Timezone); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid timezone: "+*input.Timezone)
		}
		existing.Timezone = *input.Timezone
	}

	if input.Description != nil {
		existing.Description = *input.Description
	}

	if input.Branch != nil && *input.Branch != "" {
		existing.Branch = *input.Branch
	}

	if input.EnvironmentID != nil {
		existing.EnvironmentID = input.EnvironmentID
	}

	if input.Variables != nil {
		b, _ := json.Marshal(input.Variables)
		existing.Variables = string(b)
	}

	if input.Enabled != nil {
		existing.Enabled = *input.Enabled
	}

	svc := h.scheduleService()
	if err := svc.Update(c.Context(), existing); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update schedule: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "pipeline_schedule", scheduleID, input)

	// Re-fetch to return updated data
	updated, err := h.repo.Schedules.GetByID(c.Context(), scheduleID)
	if err != nil {
		return c.JSON(existing)
	}
	return c.JSON(updated)
}

// DeleteSchedule deletes a schedule.
// DELETE /api/v1/schedules/:sid
func (h *Handler) DeleteSchedule(c fiber.Ctx) error {
	scheduleID := c.Params("sid")

	if _, err := h.repo.Schedules.GetByID(c.Context(), scheduleID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}

	if err := h.repo.Schedules.Delete(c.Context(), scheduleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete schedule")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "pipeline_schedule", scheduleID, nil)

	return c.JSON(fiber.Map{"message": "schedule deleted"})
}

// EnableSchedule enables a schedule and computes next_run_at.
// POST /api/v1/schedules/:sid/enable
func (h *Handler) EnableSchedule(c fiber.Ctx) error {
	scheduleID := c.Params("sid")

	svc := h.scheduleService()
	if err := svc.Enable(c.Context(), scheduleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enable schedule: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "enable", "pipeline_schedule", scheduleID, nil)

	updated, err := h.repo.Schedules.GetByID(c.Context(), scheduleID)
	if err != nil {
		return c.JSON(fiber.Map{"message": "schedule enabled"})
	}
	return c.JSON(updated)
}

// DisableSchedule disables a schedule and clears next_run_at.
// POST /api/v1/schedules/:sid/disable
func (h *Handler) DisableSchedule(c fiber.Ctx) error {
	scheduleID := c.Params("sid")

	svc := h.scheduleService()
	if err := svc.Disable(c.Context(), scheduleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to disable schedule: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "disable", "pipeline_schedule", scheduleID, nil)

	updated, err := h.repo.Schedules.GetByID(c.Context(), scheduleID)
	if err != nil {
		return c.JSON(fiber.Map{"message": "schedule disabled"})
	}
	return c.JSON(updated)
}

// GetNextRuns returns the next N scheduled run times for a schedule.
// GET /api/v1/schedules/:sid/next-runs?count=5
func (h *Handler) GetNextRuns(c fiber.Ctx) error {
	scheduleID := c.Params("sid")

	count := 5
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 20 {
			count = n
		}
	}

	svc := h.scheduleService()
	times, err := svc.GetNextRuns(c.Context(), scheduleID, count)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to compute next runs: "+err.Error())
	}

	// Return as ISO 8601 strings
	result := make([]string, len(times))
	for i, t := range times {
		result[i] = t.UTC().Format(time.RFC3339)
	}

	return c.JSON(result)
}
