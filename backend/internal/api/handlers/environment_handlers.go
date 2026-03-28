package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/approval"
	"github.com/oarkflow/deploy/backend/internal/models"
)

// =========================================================================
// ENVIRONMENT HANDLERS
// =========================================================================

// ListEnvironments returns all environments for a project.
func (h *Handler) ListEnvironments(c fiber.Ctx) error {
	projectID := c.Params("id")

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	envs, err := h.repo.Environments.List(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list environments")
	}
	return c.JSON(envs)
}

type createEnvironmentInput struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Description      string `json:"description"`
	URL              string `json:"url"`
	IsProduction     bool   `json:"is_production"`
	AutoDeployBranch string `json:"auto_deploy_branch"`
}

// CreateEnvironment creates a new environment for a project.
func (h *Handler) CreateEnvironment(c fiber.Ctx) error {
	projectID := c.Params("id")

	var input createEnvironmentInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	slug := input.Slug
	if slug == "" {
		slug = toSlug(input.Name)
	}

	// Check for duplicate slug
	if existing, _ := h.repo.Environments.GetBySlug(c.Context(), projectID, slug); existing != nil {
		return fiber.NewError(fiber.StatusConflict, "an environment with this slug already exists in this project")
	}

	env := &models.Environment{
		ProjectID:         projectID,
		Name:              input.Name,
		Slug:              slug,
		Description:       input.Description,
		URL:               input.URL,
		IsProduction:      input.IsProduction,
		AutoDeployBranch:  input.AutoDeployBranch,
		RequiredApprovers: "[]",
		ProtectionRules:   "{}",
	}

	if err := h.repo.Environments.Create(c.Context(), env); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create environment: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "environment", env.ID,
		fiber.Map{"name": input.Name, "project_id": projectID})

	return c.Status(fiber.StatusCreated).JSON(env)
}

type updateEnvironmentInput struct {
	Name             *string `json:"name"`
	Slug             *string `json:"slug"`
	Description      *string `json:"description"`
	URL              *string `json:"url"`
	IsProduction     *bool   `json:"is_production"`
	AutoDeployBranch *string `json:"auto_deploy_branch"`
	DeployFreeze     *bool   `json:"deploy_freeze"`
}

// UpdateEnvironment updates an environment.
func (h *Handler) UpdateEnvironment(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input updateEnvironmentInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	if input.Name != nil && *input.Name != "" {
		env.Name = *input.Name
	}
	if input.Slug != nil && *input.Slug != "" {
		// Check for duplicate slug
		if existing, _ := h.repo.Environments.GetBySlug(c.Context(), env.ProjectID, *input.Slug); existing != nil && existing.ID != envID {
			return fiber.NewError(fiber.StatusConflict, "an environment with this slug already exists in this project")
		}
		env.Slug = *input.Slug
	}
	if input.Description != nil {
		env.Description = *input.Description
	}
	if input.URL != nil {
		env.URL = *input.URL
	}
	if input.IsProduction != nil {
		env.IsProduction = *input.IsProduction
	}
	if input.AutoDeployBranch != nil {
		env.AutoDeployBranch = *input.AutoDeployBranch
	}
	if input.DeployFreeze != nil {
		env.DeployFreeze = *input.DeployFreeze
	}

	if err := h.repo.Environments.Update(c.Context(), env); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update environment")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "environment", envID, input)

	return c.JSON(env)
}

// DeleteEnvironment deletes an environment.
func (h *Handler) DeleteEnvironment(c fiber.Ctx) error {
	envID := c.Params("eid")

	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	if err := h.repo.Environments.Delete(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete environment")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "environment", envID, nil)

	return c.JSON(fiber.Map{"message": "environment deleted"})
}

type lockEnvironmentInput struct {
	Reason string `json:"reason"`
}

// LockEnvironment locks an environment to prevent deployments.
func (h *Handler) LockEnvironment(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input lockEnvironmentInput
	_ = c.Bind().JSON(&input)

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	if env.LockOwnerID != nil {
		return fiber.NewError(fiber.StatusConflict, "environment is already locked")
	}

	userID := getUserID(c)
	if err := h.repo.Environments.Lock(c.Context(), envID, userID, input.Reason); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to lock environment")
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "lock", "environment", envID,
		fiber.Map{"reason": input.Reason})

	// Return updated environment
	env, _ = h.repo.Environments.GetByID(c.Context(), envID)
	return c.JSON(env)
}

// UnlockEnvironment unlocks an environment.
func (h *Handler) UnlockEnvironment(c fiber.Ctx) error {
	envID := c.Params("eid")

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	if env.LockOwnerID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "environment is not locked")
	}

	if err := h.repo.Environments.Unlock(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to unlock environment")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "unlock", "environment", envID, nil)

	// Return updated environment
	env, _ = h.repo.Environments.GetByID(c.Context(), envID)
	return c.JSON(env)
}

// =========================================================================
// DEPLOYMENT HANDLERS
// =========================================================================

// ListDeployments returns all deployments for an environment.
func (h *Handler) ListDeployments(c fiber.Ctx) error {
	envID := c.Params("eid")

	// Verify environment exists
	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	limit, offset := h.pagination(c)
	deployments, err := h.repo.Deployments.ListByEnvironment(c.Context(), envID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list deployments")
	}
	return c.JSON(deployments)
}

type triggerDeploymentInput struct {
	PipelineID string `json:"pipeline_id"`
	Version    string `json:"version"`
	CommitSHA  string `json:"commit_sha"`
	ImageTag   string `json:"image_tag"`
}

// TriggerDeployment creates a new deployment for an environment.
func (h *Handler) TriggerDeployment(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input triggerDeploymentInput
	_ = c.Bind().JSON(&input)

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Check if environment is locked
	if env.LockOwnerID != nil {
		return fiber.NewError(fiber.StatusConflict, "environment is locked — cannot deploy")
	}

	// Check if environment is frozen
	if env.DeployFreeze {
		return fiber.NewError(fiber.StatusConflict, "environment has an active deploy freeze")
	}

	userID := getUserID(c)
	now := time.Now()

	// Set strategy from environment configuration
	strategy := env.Strategy
	if strategy == "" {
		strategy = "recreate"
	}

	dep := &models.Deployment{
		EnvironmentID:      envID,
		Version:            input.Version,
		Status:             "pending",
		CommitSHA:          input.CommitSHA,
		ImageTag:           input.ImageTag,
		DeployedBy:         userID,
		StartedAt:          &now,
		HealthCheckStatus:  "unknown",
		Metadata:           "{}",
		Strategy:           strategy,
		CanaryWeight:       0,
		HealthCheckResults: "[]",
		StrategyState:      "{}",
	}

	if input.PipelineID != "" {
		dep.PipelineRunID = &input.PipelineID
	}

	if err := h.repo.Deployments.Create(c.Context(), dep); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create deployment: "+err.Error())
	}

	// Check if environment requires approval
	approvalSvc := approval.NewService(h.repo)
	requiresApproval, rules := approvalSvc.CheckRequired(env)

	if requiresApproval {
		// Get approvers from environment
		approvers := approvalSvc.GetApprovers(env)
		if len(approvers) == 0 {
			// No approvers configured — fall through to normal deploy
			requiresApproval = false
		}
	}

	if requiresApproval {
		// Create approval request — keep deployment in "pending" status
		approvers := approvalSvc.GetApprovers(env)
		minApprovals := 1
		if rules != nil && rules.MinApprovals > 0 {
			minApprovals = rules.MinApprovals
		}

		approversJSON, _ := json.Marshal(approvers)
		_ = approversJSON

		appr, err := approvalSvc.RequestApproval(c.Context(), &approval.ApprovalRequest{
			Type:          "deployment",
			DeploymentID:  &dep.ID,
			EnvironmentID: &envID,
			ProjectID:     env.ProjectID,
			RequestedBy:   userID,
			MinApprovals:  minApprovals,
			Approvers:     approvers,
		})
		if err != nil {
			// Approval creation failed — still created deployment, but log the error
			_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "deploy", "environment", envID,
				fiber.Map{"deployment_id": dep.ID, "version": input.Version, "approval_error": err.Error()})
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{
				"deployment":        dep,
				"approval_required": true,
				"approval_error":    err.Error(),
			})
		}

		_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "deploy", "environment", envID,
			fiber.Map{"deployment_id": dep.ID, "version": input.Version, "approval_id": appr.ID, "approval_required": true})

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"deployment":        dep,
			"approval_required": true,
			"approval":          appr,
		})
	}

	// No approval required — proceed with deployment
	// Update status to deploying
	_ = h.repo.Deployments.UpdateStatus(c.Context(), dep.ID, "deploying")
	dep.Status = "deploying"

	// Update environment's current deployment
	_ = h.repo.Environments.UpdateCurrentDeployment(c.Context(), envID, dep.ID)

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "deploy", "environment", envID,
		fiber.Map{"deployment_id": dep.ID, "version": input.Version})

	return c.Status(fiber.StatusCreated).JSON(dep)
}

type rollbackDeploymentInput struct {
	DeploymentID string `json:"deployment_id"`
}

// RollbackDeployment rolls back to a previous deployment.
func (h *Handler) RollbackDeployment(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input rollbackDeploymentInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.DeploymentID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "deployment_id is required")
	}

	env, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	// Check if environment is locked
	if env.LockOwnerID != nil {
		return fiber.NewError(fiber.StatusConflict, "environment is locked — cannot rollback")
	}

	// Get the target deployment to rollback to
	targetDep, err := h.repo.Deployments.GetByID(c.Context(), input.DeploymentID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "target deployment not found")
	}

	if targetDep.EnvironmentID != envID {
		return fiber.NewError(fiber.StatusBadRequest, "target deployment does not belong to this environment")
	}

	// Mark current deployment as rolled_back
	if env.CurrentDeploymentID != nil {
		_ = h.repo.Deployments.SetFinished(c.Context(), *env.CurrentDeploymentID, "rolled_back")
	}

	userID := getUserID(c)
	now := time.Now()
	dep := &models.Deployment{
		EnvironmentID:     envID,
		Version:           targetDep.Version,
		Status:            "deploying",
		CommitSHA:         targetDep.CommitSHA,
		ImageTag:          targetDep.ImageTag,
		DeployedBy:        userID,
		StartedAt:         &now,
		HealthCheckStatus: "unknown",
		RollbackFromID:    env.CurrentDeploymentID,
		Metadata:          "{}",
	}

	if err := h.repo.Deployments.Create(c.Context(), dep); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create rollback deployment")
	}

	// Update environment's current deployment
	_ = h.repo.Environments.UpdateCurrentDeployment(c.Context(), envID, dep.ID)

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "rollback", "environment", envID,
		fiber.Map{"deployment_id": dep.ID, "rollback_to": input.DeploymentID})

	return c.Status(fiber.StatusCreated).JSON(dep)
}

type promoteDeploymentInput struct {
	SourceEnvID string `json:"source_env_id"`
}

// PromoteDeployment promotes a deployment from another environment.
func (h *Handler) PromoteDeployment(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input promoteDeploymentInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.SourceEnvID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "source_env_id is required")
	}

	// Verify target environment
	targetEnv, err := h.repo.Environments.GetByID(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "target environment not found")
	}

	// Check if target environment is locked
	if targetEnv.LockOwnerID != nil {
		return fiber.NewError(fiber.StatusConflict, "target environment is locked — cannot promote")
	}

	if targetEnv.DeployFreeze {
		return fiber.NewError(fiber.StatusConflict, "target environment has an active deploy freeze")
	}

	// Get source environment's current deployment
	sourceEnv, err := h.repo.Environments.GetByID(c.Context(), input.SourceEnvID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "source environment not found")
	}

	if sourceEnv.CurrentDeploymentID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "source environment has no current deployment")
	}

	// Verify both environments belong to the same project
	if sourceEnv.ProjectID != targetEnv.ProjectID {
		return fiber.NewError(fiber.StatusBadRequest, "source and target environments must belong to the same project")
	}

	chainEdges, err := h.repo.EnvironmentChain.ListByProject(c.Context(), targetEnv.ProjectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to validate promotion chain")
	}
	if len(chainEdges) > 0 {
		allowed, err := h.repo.EnvironmentChain.IsPromotionAllowed(c.Context(), targetEnv.ProjectID, input.SourceEnvID, envID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to validate promotion chain")
		}
		if !allowed {
			return fiber.NewError(fiber.StatusForbidden, "promotion not allowed by project environment chain")
		}
	}

	sourceDep, err := h.repo.Deployments.GetByID(c.Context(), *sourceEnv.CurrentDeploymentID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "source deployment not found")
	}

	userID := getUserID(c)
	now := time.Now()
	dep := &models.Deployment{
		EnvironmentID:     envID,
		PipelineRunID:     sourceDep.PipelineRunID,
		Version:           sourceDep.Version,
		Status:            "deploying",
		CommitSHA:         sourceDep.CommitSHA,
		ImageTag:          sourceDep.ImageTag,
		DeployedBy:        userID,
		StartedAt:         &now,
		HealthCheckStatus: "unknown",
		Metadata:          `{"promoted_from":"` + input.SourceEnvID + `"}`,
	}

	if err := h.repo.Deployments.Create(c.Context(), dep); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create promoted deployment")
	}

	// Update environment's current deployment
	_ = h.repo.Environments.UpdateCurrentDeployment(c.Context(), envID, dep.ID)

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "promote", "environment", envID,
		fiber.Map{"deployment_id": dep.ID, "source_env_id": input.SourceEnvID})

	return c.Status(fiber.StatusCreated).JSON(dep)
}

// =========================================================================
// ENVIRONMENT OVERRIDE HANDLERS
// =========================================================================

// ListEnvOverrides returns all env overrides for an environment.
func (h *Handler) ListEnvOverrides(c fiber.Ctx) error {
	envID := c.Params("eid")

	// Verify environment exists
	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	overrides, err := h.repo.EnvOverrides.ListByEnvironment(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list env overrides")
	}

	// Mask secret values
	for i := range overrides {
		if overrides[i].IsSecret {
			overrides[i].ValueEnc = "••••••••"
		}
	}

	return c.JSON(overrides)
}

type envOverrideItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

// SaveEnvOverrides replaces all env overrides for an environment.
func (h *Handler) SaveEnvOverrides(c fiber.Ctx) error {
	envID := c.Params("eid")

	var input []envOverrideItem
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body — expected array of {key, value, is_secret}")
	}

	// Verify environment exists
	if _, err := h.repo.Environments.GetByID(c.Context(), envID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "environment not found")
	}

	var overrides []models.EnvOverride
	for _, item := range input {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		overrides = append(overrides, models.EnvOverride{
			EnvironmentID: envID,
			Key:           key,
			ValueEnc:      item.Value,
			IsSecret:      item.IsSecret,
		})
	}

	if err := h.repo.EnvOverrides.BulkSave(c.Context(), envID, overrides); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save env overrides")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "save_overrides", "environment", envID,
		fiber.Map{"count": len(overrides)})

	// Return saved overrides
	saved, err := h.repo.EnvOverrides.ListByEnvironment(c.Context(), envID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "saved but failed to reload")
	}

	// Mask secret values
	for i := range saved {
		if saved[i].IsSecret {
			saved[i].ValueEnc = "••••••••"
		}
	}

	return c.JSON(saved)
}

// =========================================================================
// RECENT DEPLOYMENTS (for dashboard)
// =========================================================================

// ListRecentDeployments returns the most recent deployments across all environments.
func (h *Handler) ListRecentDeployments(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	deployments, err := h.repo.Deployments.ListAll(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list recent deployments")
	}
	return c.JSON(deployments)
}
