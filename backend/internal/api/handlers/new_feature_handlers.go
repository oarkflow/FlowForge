package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/admin"
	"github.com/oarkflow/deploy/backend/internal/engine/queue"
	"github.com/oarkflow/deploy/backend/internal/features"
	"github.com/oarkflow/deploy/backend/internal/models"
	securitypkg "github.com/oarkflow/deploy/backend/internal/security"
	"github.com/oarkflow/deploy/backend/internal/templates"
)

// Extended handler fields (set via SetXxx methods to avoid changing Handler constructor)
var (
	templateStore  *templates.Store
	flagService    *features.FlagService
	deadLetterQ    *queue.DeadLetterQueue
	backupService  *admin.BackupService
	scanService    *securitypkg.ScanService
)

// SetTemplateStore wires the template store into the handler.
func SetTemplateStore(ts *templates.Store) { templateStore = ts }

// SetFlagService wires the feature flag service into the handler.
func SetFlagService(fs *features.FlagService) { flagService = fs }

// SetDeadLetterQueue wires the DLQ into the handler.
func SetDeadLetterQueue(dlq *queue.DeadLetterQueue) { deadLetterQ = dlq }

// SetBackupService wires the backup service into the handler.
func SetBackupService(bs *admin.BackupService) { backupService = bs }

// SetScanService wires the security scan service into the handler.
func SetScanService(ss *securitypkg.ScanService) { scanService = ss }

// =========================================================================
// TEMPLATE HANDLERS
// =========================================================================

func (h *Handler) ListTemplates(c fiber.Ctx) error {
	if templateStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "template store not configured")
	}
	limit, offset := h.pagination(c)
	category := c.Query("category")
	builtinOnly := c.Query("builtin") == "true"

	tmps, err := templateStore.List(c.Context(), category, builtinOnly, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list templates: "+err.Error())
	}
	return c.JSON(fiber.Map{"templates": tmps})
}

func (h *Handler) GetTemplate(c fiber.Ctx) error {
	if templateStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "template store not configured")
	}
	id := c.Params("id")
	t, err := templateStore.Get(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.JSON(t)
}

type createTemplateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Config      string `json:"config"`
	IsPublic    int    `json:"is_public"`
}

func (h *Handler) CreateTemplate(c fiber.Ctx) error {
	if templateStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "template store not configured")
	}
	var input createTemplateInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.Config == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and config are required")
	}

	t := &models.PipelineTemplate{
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		Config:      input.Config,
		IsPublic:    input.IsPublic,
		Author:      getUserID(c),
	}
	if t.Category == "" {
		t.Category = "general"
	}

	if err := templateStore.Create(c.Context(), t); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create template: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "template", t.ID, nil)
	return c.Status(fiber.StatusCreated).JSON(t)
}

func (h *Handler) UpdateTemplate(c fiber.Ctx) error {
	if templateStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "template store not configured")
	}
	id := c.Params("id")
	var input createTemplateInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	t := &models.PipelineTemplate{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		Config:      input.Config,
		IsPublic:    input.IsPublic,
		Author:      getUserID(c),
	}

	if err := templateStore.Update(c.Context(), t); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "template", id, nil)
	return c.JSON(t)
}

func (h *Handler) DeleteTemplate(c fiber.Ctx) error {
	if templateStore == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "template store not configured")
	}
	id := c.Params("id")
	if err := templateStore.Delete(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "template", id, nil)
	return c.JSON(fiber.Map{"message": "template deleted"})
}

// =========================================================================
// FEATURE FLAG HANDLERS
// =========================================================================

func (h *Handler) ListFeatureFlags(c fiber.Ctx) error {
	if flagService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "feature flags not configured")
	}
	limit, offset := h.pagination(c)
	flags, err := flagService.List(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list feature flags: "+err.Error())
	}
	return c.JSON(fiber.Map{"features": flags})
}

type updateFeatureFlagInput struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Enabled           *int   `json:"enabled"`
	RolloutPercentage *int   `json:"rollout_percentage"`
	TargetUsers       string `json:"target_users"`
	TargetOrgs        string `json:"target_orgs"`
}

func (h *Handler) UpdateFeatureFlag(c fiber.Ctx) error {
	if flagService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "feature flags not configured")
	}
	id := c.Params("id")
	var input updateFeatureFlagInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	existing, err := h.repo.FeatureFlags.GetByID(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "feature flag not found")
	}

	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Description != "" {
		existing.Description = input.Description
	}
	if input.Enabled != nil {
		existing.Enabled = *input.Enabled
	}
	if input.RolloutPercentage != nil {
		existing.RolloutPercentage = *input.RolloutPercentage
	}
	if input.TargetUsers != "" {
		existing.TargetUsers = input.TargetUsers
	}
	if input.TargetOrgs != "" {
		existing.TargetOrgs = input.TargetOrgs
	}

	if err := flagService.Update(c.Context(), existing); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update feature flag: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "feature_flag", id, nil)
	return c.JSON(existing)
}

func (h *Handler) CreateFeatureFlag(c fiber.Ctx) error {
	if flagService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "feature flags not configured")
	}
	var input updateFeatureFlagInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	enabled := 0
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	rollout := 100
	if input.RolloutPercentage != nil {
		rollout = *input.RolloutPercentage
	}
	targetUsers := "[]"
	if input.TargetUsers != "" {
		targetUsers = input.TargetUsers
	}
	targetOrgs := "[]"
	if input.TargetOrgs != "" {
		targetOrgs = input.TargetOrgs
	}

	flag := &models.FeatureFlag{
		Name:              input.Name,
		Description:       input.Description,
		Enabled:           enabled,
		RolloutPercentage: rollout,
		TargetUsers:       targetUsers,
		TargetOrgs:        targetOrgs,
	}

	if err := flagService.Create(c.Context(), flag); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create feature flag: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "feature_flag", flag.ID, nil)
	return c.Status(fiber.StatusCreated).JSON(flag)
}

// =========================================================================
// DEAD-LETTER QUEUE HANDLERS
// =========================================================================

func (h *Handler) ListDLQ(c fiber.Ctx) error {
	if deadLetterQ == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "dead-letter queue not configured")
	}
	limit, offset := h.pagination(c)
	status := c.Query("status", "pending")

	items, err := deadLetterQ.List(c.Context(), status, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list DLQ: "+err.Error())
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) RetryDLQ(c fiber.Ctx) error {
	if deadLetterQ == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "dead-letter queue not configured")
	}
	id := c.Params("id")
	job, err := deadLetterQ.Retry(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to retry DLQ item: "+err.Error())
	}
	if job == nil {
		return fiber.NewError(fiber.StatusBadRequest, "item cannot be retried (max retries reached or already retried)")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "retry", "dlq_item", id, nil)
	return c.JSON(fiber.Map{"message": "item queued for retry", "job_id": job.ID})
}

func (h *Handler) PurgeDLQ(c fiber.Ctx) error {
	if deadLetterQ == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "dead-letter queue not configured")
	}
	id := c.Params("id")
	if err := deadLetterQ.Purge(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to purge DLQ item: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "purge", "dlq_item", id, nil)
	return c.JSON(fiber.Map{"message": "item purged"})
}

// =========================================================================
// BACKUP HANDLERS
// =========================================================================

func (h *Handler) CreateBackup(c fiber.Ctx) error {
	if backupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "backup service not configured")
	}
	backup, err := backupService.CreateBackup(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "backup failed: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "backup", backup.ID, nil)
	return c.Status(fiber.StatusCreated).JSON(backup)
}

func (h *Handler) ListBackups(c fiber.Ctx) error {
	if backupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "backup service not configured")
	}
	backups, err := backupService.ListBackups()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list backups: "+err.Error())
	}
	return c.JSON(fiber.Map{"backups": backups})
}

type restoreInput struct {
	BackupID string `json:"backup_id"`
}

func (h *Handler) RestoreBackup(c fiber.Ctx) error {
	if backupService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "backup service not configured")
	}
	var input restoreInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.BackupID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "backup_id is required")
	}

	backups, err := backupService.ListBackups()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var backupPath string
	for _, b := range backups {
		if b.ID == input.BackupID || b.Filename == input.BackupID {
			backupPath = b.Path
			break
		}
	}
	if backupPath == "" {
		return fiber.NewError(fiber.StatusNotFound, "backup not found")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "restore", "backup", input.BackupID, nil)

	if err := backupService.RestoreBackup(c.Context(), backupPath); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "restore failed: "+err.Error())
	}

	return c.JSON(fiber.Map{"message": "database restored — server restart required"})
}

// =========================================================================
// SECURITY SCAN HANDLERS
// =========================================================================

func (h *Handler) GetRunSecurityResults(c fiber.Ctx) error {
	if scanService == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "scan service not configured")
	}
	runID := c.Params("rid")
	results, err := scanService.GetByRunID(c.Context(), runID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get scan results: "+err.Error())
	}
	return c.JSON(fiber.Map{"scan_results": results})
}
