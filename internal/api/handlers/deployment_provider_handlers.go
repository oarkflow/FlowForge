package handlers

import (
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// =========================================================================
// PROJECT DEPLOYMENT PROVIDER HANDLERS
// =========================================================================

type createDeploymentProviderInput struct {
	Name         string `json:"name"`
	ProviderType string `json:"provider_type"`
	Config       any    `json:"config"`
	IsActive     *int   `json:"is_active"`
}

type updateDeploymentProviderInput struct {
	Name         *string `json:"name"`
	ProviderType *string `json:"provider_type"`
	Config       any     `json:"config"`
	IsActive     *int    `json:"is_active"`
}

// ListDeploymentProviders returns all configured deployment providers for a project.
func (h *Handler) ListDeploymentProviders(c fiber.Ctx) error {
	projectID := c.Params("id")

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	limit, offset := h.pagination(c)
	providers, err := h.repo.DeploymentProviders.ListByProject(c.Context(), projectID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list deployment providers")
	}

	out := make([]fiber.Map, 0, len(providers))
	for i := range providers {
		item, err := h.sanitizeDeploymentProvider(&providers[i])
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to render deployment provider")
		}
		out = append(out, item)
	}

	return c.JSON(out)
}

// CreateDeploymentProvider creates a deployment provider for a project.
func (h *Handler) CreateDeploymentProvider(c fiber.Ctx) error {
	projectID := c.Params("id")

	var input createDeploymentProviderInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	name := strings.TrimSpace(input.Name)
	providerType := strings.ToLower(strings.TrimSpace(input.ProviderType))
	if name == "" || providerType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and provider_type are required")
	}

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	if existing, err := h.repo.DeploymentProviders.GetByName(c.Context(), projectID, name); err == nil && existing != nil {
		return fiber.NewError(fiber.StatusConflict, "deployment provider name already exists in this project")
	}

	configMap, err := normalizeConfigMap(input.Config)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "config must be a JSON object")
	}
	if err := validateProviderConfig(providerType, configMap); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	configJSON, _ := json.Marshal(configMap)
	encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "server encryption key misconfigured")
	}
	configEnc, err := crypto.Encrypt(encKey, string(configJSON))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt provider config")
	}

	isActive := 1
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	userID := getUserID(c)
	provider := &models.ProjectDeploymentProvider{
		ProjectID:    projectID,
		Name:         name,
		ProviderType: providerType,
		ConfigEnc:    configEnc,
		IsActive:     isActive,
		CreatedBy:    strPtrOrNil(userID),
	}
	if err := h.repo.DeploymentProviders.Create(c.Context(), provider); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fiber.NewError(fiber.StatusConflict, "deployment provider name already exists in this project")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create deployment provider")
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "deployment_provider", provider.ID,
		fiber.Map{"project_id": projectID, "provider_type": providerType, "name": name})

	out, err := h.sanitizeDeploymentProvider(provider)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to render deployment provider")
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

// UpdateDeploymentProvider updates a deployment provider config.
func (h *Handler) UpdateDeploymentProvider(c fiber.Ctx) error {
	projectID := c.Params("id")
	providerID := c.Params("dpid")

	var input updateDeploymentProviderInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	provider, err := h.repo.DeploymentProviders.GetByID(c.Context(), providerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment provider not found")
	}
	if provider.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment provider does not belong to this project")
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name cannot be empty")
		}
		if existing, err := h.repo.DeploymentProviders.GetByName(c.Context(), projectID, name); err == nil && existing != nil && existing.ID != providerID {
			return fiber.NewError(fiber.StatusConflict, "deployment provider name already exists in this project")
		}
		provider.Name = name
	}

	if input.ProviderType != nil {
		pType := strings.ToLower(strings.TrimSpace(*input.ProviderType))
		if pType == "" {
			return fiber.NewError(fiber.StatusBadRequest, "provider_type cannot be empty")
		}
		provider.ProviderType = pType
	}

	if input.Config != nil {
		configMap, err := normalizeConfigMap(input.Config)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "config must be a JSON object")
		}
		if err := validateProviderConfig(provider.ProviderType, configMap); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		configJSON, _ := json.Marshal(configMap)
		encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "server encryption key misconfigured")
		}
		configEnc, err := crypto.Encrypt(encKey, string(configJSON))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt provider config")
		}
		provider.ConfigEnc = configEnc
	}

	if input.IsActive != nil {
		provider.IsActive = *input.IsActive
	}

	if err := h.repo.DeploymentProviders.Update(c.Context(), provider); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fiber.NewError(fiber.StatusConflict, "deployment provider name already exists in this project")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update deployment provider")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "deployment_provider", providerID, nil)

	out, err := h.sanitizeDeploymentProvider(provider)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to render deployment provider")
	}
	return c.JSON(out)
}

// DeleteDeploymentProvider deletes a project deployment provider.
func (h *Handler) DeleteDeploymentProvider(c fiber.Ctx) error {
	projectID := c.Params("id")
	providerID := c.Params("dpid")

	provider, err := h.repo.DeploymentProviders.GetByID(c.Context(), providerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment provider not found")
	}
	if provider.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment provider does not belong to this project")
	}

	if err := h.repo.DeploymentProviders.Delete(c.Context(), providerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete deployment provider")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "deployment_provider", providerID, nil)
	return c.JSON(fiber.Map{"message": "deployment provider deleted"})
}

// TestDeploymentProvider performs low-effort config validation/decryption checks.
func (h *Handler) TestDeploymentProvider(c fiber.Ctx) error {
	projectID := c.Params("id")
	providerID := c.Params("dpid")

	provider, err := h.repo.DeploymentProviders.GetByID(c.Context(), providerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "deployment provider not found")
	}
	if provider.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "deployment provider does not belong to this project")
	}

	configMap, err := h.decryptProviderConfig(provider.ConfigEnc)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "message": "failed to decrypt provider config"})
	}
	if err := validateProviderConfig(provider.ProviderType, configMap); err != nil {
		return c.JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "deployment provider configuration is valid"})
}

// =========================================================================
// PROJECT ENVIRONMENT CHAIN HANDLERS
// =========================================================================

type environmentChainEdgeInput struct {
	SourceEnvironmentID string `json:"source_environment_id"`
	TargetEnvironmentID string `json:"target_environment_id"`
	Position            int    `json:"position"`
}

// GetEnvironmentChain returns project-level promotion edges.
func (h *Handler) GetEnvironmentChain(c fiber.Ctx) error {
	projectID := c.Params("id")

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	edges, err := h.repo.EnvironmentChain.ListByProject(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch environment chain")
	}
	return c.JSON(edges)
}

// UpdateEnvironmentChain replaces project-level promotion edges.
func (h *Handler) UpdateEnvironmentChain(c fiber.Ctx) error {
	projectID := c.Params("id")

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	var input []environmentChainEdgeInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body — expected array of chain edges")
	}

	edges := make([]models.ProjectEnvironmentChainEdge, 0, len(input))
	seen := map[string]struct{}{}
	for i, item := range input {
		sourceID := strings.TrimSpace(item.SourceEnvironmentID)
		targetID := strings.TrimSpace(item.TargetEnvironmentID)
		if sourceID == "" || targetID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "source_environment_id and target_environment_id are required")
		}
		if sourceID == targetID {
			return fiber.NewError(fiber.StatusBadRequest, "source_environment_id and target_environment_id must be different")
		}

		sourceEnv, err := h.repo.Environments.GetByID(c.Context(), sourceID)
		if err != nil || sourceEnv.ProjectID != projectID {
			return fiber.NewError(fiber.StatusBadRequest, "source environment must belong to this project")
		}
		targetEnv, err := h.repo.Environments.GetByID(c.Context(), targetID)
		if err != nil || targetEnv.ProjectID != projectID {
			return fiber.NewError(fiber.StatusBadRequest, "target environment must belong to this project")
		}

		edgeKey := sourceID + "->" + targetID
		if _, ok := seen[edgeKey]; ok {
			return fiber.NewError(fiber.StatusBadRequest, "duplicate source->target edge in request")
		}
		seen[edgeKey] = struct{}{}

		position := item.Position
		if position < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "position cannot be negative")
		}
		if position == 0 {
			position = i
		}

		edges = append(edges, models.ProjectEnvironmentChainEdge{
			SourceEnvironmentID: sourceID,
			TargetEnvironmentID: targetID,
			Position:            position,
		})
	}

	if err := h.repo.EnvironmentChain.ReplaceForProject(c.Context(), projectID, edges); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update environment chain")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "environment_chain", projectID,
		fiber.Map{"edges": len(edges)})

	saved, err := h.repo.EnvironmentChain.ListByProject(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "updated but failed to reload environment chain")
	}
	return c.JSON(saved)
}

// =========================================================================
// PIPELINE STAGE -> ENVIRONMENT MAPPING HANDLERS
// =========================================================================

type stageEnvironmentMappingInput struct {
	StageName     string `json:"stage_name"`
	EnvironmentID string `json:"environment_id"`
}

// GetPipelineStageEnvironmentMappings returns mappings for a pipeline.
func (h *Handler) GetPipelineStageEnvironmentMappings(c fiber.Ctx) error {
	projectID := c.Params("id")
	pipelineID := c.Params("pid")

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	pipeline, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}
	if pipeline.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "pipeline does not belong to this project")
	}

	mappings, err := h.repo.StageEnvironmentMappings.ListByPipeline(c.Context(), projectID, pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list stage environment mappings")
	}
	return c.JSON(mappings)
}

// UpdatePipelineStageEnvironmentMappings replaces mappings for a pipeline.
func (h *Handler) UpdatePipelineStageEnvironmentMappings(c fiber.Ctx) error {
	projectID := c.Params("id")
	pipelineID := c.Params("pid")

	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	pipeline, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}
	if pipeline.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "pipeline does not belong to this project")
	}

	var input []stageEnvironmentMappingInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body — expected array of stage mappings")
	}

	mappings := make([]models.PipelineStageEnvironmentMapping, 0, len(input))
	seenStage := map[string]struct{}{}
	for _, item := range input {
		stage := strings.TrimSpace(item.StageName)
		envID := strings.TrimSpace(item.EnvironmentID)
		if stage == "" || envID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "stage_name and environment_id are required")
		}
		if _, ok := seenStage[stage]; ok {
			return fiber.NewError(fiber.StatusBadRequest, "duplicate stage_name in request")
		}
		seenStage[stage] = struct{}{}

		env, err := h.repo.Environments.GetByID(c.Context(), envID)
		if err != nil || env.ProjectID != projectID {
			return fiber.NewError(fiber.StatusBadRequest, "environment must belong to this project")
		}

		mappings = append(mappings, models.PipelineStageEnvironmentMapping{
			StageName:     stage,
			EnvironmentID: envID,
		})
	}

	if err := h.repo.StageEnvironmentMappings.ReplaceForPipeline(c.Context(), projectID, pipelineID, mappings); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update stage environment mappings")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "stage_environment_mappings", pipelineID,
		fiber.Map{"count": len(mappings), "project_id": projectID})

	saved, err := h.repo.StageEnvironmentMappings.ListByPipeline(c.Context(), projectID, pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "updated but failed to reload stage environment mappings")
	}
	return c.JSON(saved)
}

// =========================================================================
// Helpers
// =========================================================================

func normalizeConfigMap(raw any) (map[string]any, error) {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "config must be an object")
	}
	return m, nil
}

func validateProviderConfig(providerType string, config map[string]any) error {
	switch providerType {
	case "aws":
		return validateAWSProviderConfig(config)
	default:
		// Extensible: unknown providers are accepted without strict schema for now.
		return nil
	}
}

func validateAWSProviderConfig(config map[string]any) error {
	region := getMapString(config, "region")
	if strings.TrimSpace(region) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "aws config requires region")
	}

	authMode := strings.ToLower(strings.TrimSpace(getMapString(config, "auth_mode")))
	if authMode == "" {
		return fiber.NewError(fiber.StatusBadRequest, "aws config requires auth_mode")
	}

	switch authMode {
	case "access_key":
		if strings.TrimSpace(getMapString(config, "access_key_id")) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "aws access_key auth_mode requires access_key_id")
		}
		if strings.TrimSpace(getMapString(config, "secret_access_key")) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "aws access_key auth_mode requires secret_access_key")
		}
	case "assume_role":
		if strings.TrimSpace(getMapString(config, "role_arn")) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "aws assume_role auth_mode requires role_arn")
		}
	case "default":
		// No additional required fields.
	default:
		return fiber.NewError(fiber.StatusBadRequest, "aws auth_mode must be one of: access_key, assume_role, default")
	}

	return nil
}

func getMapString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func (h *Handler) decryptProviderConfig(configEnc string) (map[string]any, error) {
	encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	plain, err := crypto.Decrypt(encKey, configEnc)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{}
	if err := json.Unmarshal([]byte(plain), &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (h *Handler) sanitizeDeploymentProvider(provider *models.ProjectDeploymentProvider) (fiber.Map, error) {
	cfg, err := h.decryptProviderConfig(provider.ConfigEnc)
	if err != nil {
		cfg = map[string]any{}
	}
	masked := maskProviderConfig(provider.ProviderType, cfg)

	return fiber.Map{
		"id":            provider.ID,
		"project_id":    provider.ProjectID,
		"name":          provider.Name,
		"provider_type": provider.ProviderType,
		"is_active":     provider.IsActive,
		"created_by":    provider.CreatedBy,
		"created_at":    provider.CreatedAt,
		"updated_at":    provider.UpdatedAt,
		"config":        masked,
	}, nil
}

func maskProviderConfig(providerType string, cfg map[string]any) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	masked := make(map[string]any, len(cfg))
	for k, v := range cfg {
		masked[k] = v
	}

	// Never return raw secret-like fields.
	for key := range masked {
		lk := strings.ToLower(key)
		if strings.Contains(lk, "secret") || strings.Contains(lk, "token") || strings.Contains(lk, "password") || strings.Contains(lk, "private_key") {
			masked[key] = "••••••••"
		}
	}

	if providerType == "aws" {
		if _, ok := masked["secret_access_key"]; ok {
			masked["secret_access_key"] = "••••••••"
		}
		if _, ok := masked["session_token"]; ok {
			masked["session_token"] = "••••••••"
		}
	}

	return masked
}
