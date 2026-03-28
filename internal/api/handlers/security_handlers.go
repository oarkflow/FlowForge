package handlers

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/secrets"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// =========================================================================
// SECRET PROVIDER HANDLERS
// =========================================================================

// ListSecretProviders returns all configured external secret providers for a project.
func (h *Handler) ListSecretProviders(c fiber.Ctx) error {
	projectID := c.Params("id")
	limit, offset := h.pagination(c)

	providers, err := h.repo.SecretProviders.ListByProject(c.Context(), projectID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list secret providers")
	}
	return c.JSON(providers)
}

type createSecretProviderInput struct {
	Name         string `json:"name" validate:"required"`
	ProviderType string `json:"provider_type" validate:"required,oneof=vault aws gcp"`
	Config       any    `json:"config" validate:"required"`
	Priority     int    `json:"priority"`
}

// CreateSecretProvider registers a new external secret provider for a project.
func (h *Handler) CreateSecretProvider(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input createSecretProviderInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" || input.ProviderType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and provider_type are required")
	}

	// Verify project exists.
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	// Encrypt the config JSON.
	configJSON, err := json.Marshal(input.Config)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid config object")
	}

	encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "server encryption key misconfigured")
	}

	encrypted, err := crypto.Encrypt(encKey, string(configJSON))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt provider config")
	}

	userID := getUserID(c)
	provider := &models.SecretProviderConfig{
		ProjectID:    &projectID,
		Name:         input.Name,
		ProviderType: input.ProviderType,
		ConfigEnc:    encrypted,
		IsActive:     1,
		Priority:     input.Priority,
		CreatedBy:    &userID,
	}

	if err := h.repo.SecretProviders.Create(c.Context(), provider); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create secret provider: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "secret_provider", provider.ID,
		fiber.Map{"project_id": projectID, "provider_type": input.ProviderType, "name": input.Name})

	return c.Status(fiber.StatusCreated).JSON(provider)
}

type updateSecretProviderInput struct {
	Name     *string `json:"name"`
	Config   any     `json:"config"`
	IsActive *int    `json:"is_active"`
	Priority *int    `json:"priority"`
}

// UpdateSecretProvider modifies an existing secret provider configuration.
func (h *Handler) UpdateSecretProvider(c fiber.Ctx) error {
	providerID := c.Params("spid")
	var input updateSecretProviderInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	provider, err := h.repo.SecretProviders.GetByID(c.Context(), providerID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "secret provider not found")
	}

	if input.Name != nil && *input.Name != "" {
		provider.Name = *input.Name
	}
	if input.IsActive != nil {
		provider.IsActive = *input.IsActive
	}
	if input.Priority != nil {
		provider.Priority = *input.Priority
	}
	if input.Config != nil {
		configJSON, err := json.Marshal(input.Config)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid config object")
		}
		encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "server encryption key misconfigured")
		}
		encrypted, err := crypto.Encrypt(encKey, string(configJSON))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt provider config")
		}
		provider.ConfigEnc = encrypted
	}

	if err := h.repo.SecretProviders.Update(c.Context(), provider); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update secret provider")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "secret_provider", providerID, nil)

	return c.JSON(provider)
}

// DeleteSecretProvider removes a secret provider configuration.
func (h *Handler) DeleteSecretProvider(c fiber.Ctx) error {
	providerID := c.Params("spid")

	if err := h.repo.SecretProviders.Delete(c.Context(), providerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete secret provider")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "secret_provider", providerID, nil)

	return c.JSON(fiber.Map{"message": "secret provider deleted"})
}

// GetSecretProviderHealth checks the health of all configured secret providers for a project.
func (h *Handler) GetSecretProviderHealth(c fiber.Ctx) error {
	projectID := c.Params("id")

	providers, err := h.repo.SecretProviders.ListActive(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list providers")
	}

	results := make([]fiber.Map, 0, len(providers))
	for _, p := range providers {
		results = append(results, fiber.Map{
			"id":            p.ID,
			"name":          p.Name,
			"provider_type": p.ProviderType,
			"is_active":     p.IsActive,
			"priority":      p.Priority,
		})
	}

	return c.JSON(fiber.Map{
		"providers": results,
		"count":     len(results),
	})
}

// =========================================================================
// SECRET ROTATION HANDLERS
// =========================================================================

// ListRotationStatus returns rotation status for all secrets in a project
// that have a rotation interval configured.
func (h *Handler) ListRotationStatus(c fiber.Ctx) error {
	projectID := c.Params("id")

	statuses, err := h.rotationTracker.ListAll(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list rotation status")
	}
	return c.JSON(statuses)
}

// ListOverdueSecrets returns secrets that are past their rotation deadline.
func (h *Handler) ListOverdueSecrets(c fiber.Ctx) error {
	overdue, err := h.rotationTracker.ListOverdue(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list overdue secrets")
	}
	return c.JSON(overdue)
}

type setRotationPolicyInput struct {
	Interval    string `json:"interval" validate:"required"`
	MarkRotated bool   `json:"mark_rotated"`
}

// SetSecretRotationPolicy configures the rotation interval for a secret.
func (h *Handler) SetSecretRotationPolicy(c fiber.Ctx) error {
	secretID := c.Params("secretId")
	var input setRotationPolicyInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Interval == "" {
		return fiber.NewError(fiber.StatusBadRequest, "interval is required (e.g. '30d', '90d', '365d')")
	}

	if err := h.rotationTracker.SetRotationPolicy(c.Context(), secretID, input.Interval, input.MarkRotated); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to set rotation policy: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "secret_rotation", secretID,
		fiber.Map{"interval": input.Interval, "mark_rotated": input.MarkRotated})

	return c.JSON(fiber.Map{"message": "rotation policy updated", "secret_id": secretID, "interval": input.Interval})
}

// MarkSecretRotated records that a secret has been rotated right now.
func (h *Handler) MarkSecretRotated(c fiber.Ctx) error {
	secretID := c.Params("secretId")

	if err := h.rotationTracker.MarkRotated(c.Context(), secretID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to mark rotated: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "rotate", "secret", secretID, nil)

	return c.JSON(fiber.Map{"message": "secret marked as rotated", "secret_id": secretID})
}

// =========================================================================
// IP ALLOWLIST HANDLERS
// =========================================================================

// ListIPAllowlist returns IP allowlist entries, optionally filtered by project.
func (h *Handler) ListIPAllowlist(c fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID != "" {
		entries, err := h.repo.IPAllowlist.ListByProject(c.Context(), projectID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to list IP allowlist")
		}
		return c.JSON(entries)
	}
	entries, err := h.repo.IPAllowlist.ListAll(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list IP allowlist")
	}
	return c.JSON(entries)
}

// ListGlobalIPAllowlist returns global IP allowlist entries only.
func (h *Handler) ListGlobalIPAllowlist(c fiber.Ctx) error {
	entries, err := h.repo.IPAllowlist.ListGlobal(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list global IP allowlist")
	}
	return c.JSON(entries)
}

type createIPAllowlistInput struct {
	CIDR  string `json:"cidr" validate:"required"`
	Scope string `json:"scope" validate:"required,oneof=global project"`
	Label string `json:"label"`
}

// CreateIPAllowlistEntry adds a new CIDR entry to the allowlist.
func (h *Handler) CreateIPAllowlistEntry(c fiber.Ctx) error {
	projectID := c.Params("id") // may be empty for global entries
	var input createIPAllowlistInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.CIDR == "" {
		return fiber.NewError(fiber.StatusBadRequest, "cidr is required")
	}
	if input.Scope == "" {
		input.Scope = "project"
	}

	userID := getUserID(c)
	entry := &models.IPAllowlistEntry{
		Scope:     input.Scope,
		CIDR:      input.CIDR,
		Label:     input.Label,
		CreatedBy: &userID,
	}
	if input.Scope == "project" && projectID != "" {
		entry.ProjectID = &projectID
	}

	if err := h.repo.IPAllowlist.Create(c.Context(), entry); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create IP allowlist entry: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "ip_allowlist", entry.ID,
		fiber.Map{"cidr": input.CIDR, "scope": input.Scope, "project_id": projectID})

	return c.Status(fiber.StatusCreated).JSON(entry)
}

// DeleteIPAllowlistEntry removes an IP allowlist entry.
func (h *Handler) DeleteIPAllowlistEntry(c fiber.Ctx) error {
	entryID := c.Params("aid")

	if err := h.repo.IPAllowlist.Delete(c.Context(), entryID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete IP allowlist entry")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "ip_allowlist", entryID, nil)

	return c.JSON(fiber.Map{"message": "IP allowlist entry deleted"})
}

// =========================================================================
// SECRET SCANNER HANDLERS
// =========================================================================

type scanRepoInput struct {
	Path          string  `json:"path" validate:"required"`
	MinConfidence float64 `json:"min_confidence"`
}

// ScanRepositoryForSecrets scans a repository directory for hardcoded secrets.
func (h *Handler) ScanRepositoryForSecrets(c fiber.Ctx) error {
	var input scanRepoInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Path == "" {
		return fiber.NewError(fiber.StatusBadRequest, "path is required")
	}

	findings, err := h.secretScanner.ScanDirectory(input.Path)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to scan repository: "+err.Error())
	}

	// Filter by confidence threshold if provided.
	minConf := input.MinConfidence
	if minConf <= 0 {
		minConf = 0.50
	}
	var filtered []secrets.ScanFinding
	for _, f := range findings {
		if f.Confidence >= minConf {
			filtered = append(filtered, f)
		}
	}

	return c.JSON(fiber.Map{
		"findings":       filtered,
		"total":          len(filtered),
		"min_confidence": minConf,
	})
}

// ScanTextForSecrets scans raw text content (e.g. pipeline config or env file)
// for hardcoded secrets without requiring a file path.
func (h *Handler) ScanTextForSecrets(c fiber.Ctx) error {
	type scanTextInput struct {
		Content       string  `json:"content" validate:"required"`
		MinConfidence float64 `json:"min_confidence"`
	}
	var input scanTextInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Content == "" {
		return fiber.NewError(fiber.StatusBadRequest, "content is required")
	}

	// Create a temporary directory with a single file to re-use the scanner.
	tmpDir, err := os.MkdirTemp("", "flowforge-scan-*")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "content.txt")
	if err := os.WriteFile(tmpFile, []byte(input.Content), 0600); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to write temp file")
	}

	findings, err := h.secretScanner.ScanDirectory(tmpDir)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to scan content: "+err.Error())
	}

	minConf := input.MinConfidence
	if minConf <= 0 {
		minConf = 0.50
	}
	var filtered []secrets.ScanFinding
	for _, f := range findings {
		if f.Confidence >= minConf {
			filtered = append(filtered, f)
		}
	}

	return c.JSON(fiber.Map{
		"findings": filtered,
		"total":    len(filtered),
	})
}
