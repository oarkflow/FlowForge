package handlers

import (
	"encoding/hex"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/registry"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// =========================================================================
// REGISTRY HANDLERS
// =========================================================================

// ListRegistries returns all registries for a project.
func (h *Handler) ListRegistries(c fiber.Ctx) error {
	projectID := c.Params("id")

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	registries, err := h.repo.Registries.List(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list registries")
	}
	return c.JSON(registries)
}

type createRegistryInput struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsDefault bool   `json:"is_default"`
}

// CreateRegistry creates a new registry for a project.
func (h *Handler) CreateRegistry(c fiber.Ctx) error {
	projectID := c.Params("id")

	var input createRegistryInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if input.Type == "" {
		return fiber.NewError(fiber.StatusBadRequest, "type is required")
	}

	validTypes := map[string]bool{
		"dockerhub": true, "ecr": true, "gcr": true,
		"acr": true, "harbor": true, "ghcr": true, "generic": true,
	}
	if !validTypes[input.Type] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid registry type")
	}

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	// Encrypt credentials
	var credentialsEnc string
	if input.Password != "" {
		encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "encryption configuration error")
		}
		encrypted, err := crypto.Encrypt(encKey, input.Password)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt credentials")
		}
		credentialsEnc = encrypted
	}

	reg := &models.Registry{
		ProjectID:      projectID,
		Name:           input.Name,
		Type:           input.Type,
		URL:            input.URL,
		Username:       input.Username,
		CredentialsEnc: credentialsEnc,
		IsDefault:      input.IsDefault,
	}

	if err := h.repo.Registries.Create(c.Context(), reg); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create registry: "+err.Error())
	}

	// If this is the default, unset others
	if input.IsDefault {
		_ = h.repo.Registries.SetDefault(c.Context(), projectID, reg.ID)
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "registry", reg.ID,
		fiber.Map{"name": input.Name, "type": input.Type, "project_id": projectID})

	return c.Status(fiber.StatusCreated).JSON(reg)
}

type updateRegistryInput struct {
	Name      *string `json:"name"`
	Type      *string `json:"type"`
	URL       *string `json:"url"`
	Username  *string `json:"username"`
	Password  *string `json:"password"`
	IsDefault *bool   `json:"is_default"`
}

// UpdateRegistry updates a registry.
func (h *Handler) UpdateRegistry(c fiber.Ctx) error {
	registryID := c.Params("rid")

	var input updateRegistryInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	if input.Name != nil && *input.Name != "" {
		reg.Name = *input.Name
	}
	if input.Type != nil && *input.Type != "" {
		validTypes := map[string]bool{
			"dockerhub": true, "ecr": true, "gcr": true,
			"acr": true, "harbor": true, "ghcr": true, "generic": true,
		}
		if !validTypes[*input.Type] {
			return fiber.NewError(fiber.StatusBadRequest, "invalid registry type")
		}
		reg.Type = *input.Type
	}
	if input.URL != nil {
		reg.URL = *input.URL
	}
	if input.Username != nil {
		reg.Username = *input.Username
	}

	// Re-encrypt credentials if password is provided
	if input.Password != nil && *input.Password != "" {
		encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "encryption configuration error")
		}
		encrypted, err := crypto.Encrypt(encKey, *input.Password)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt credentials")
		}
		reg.CredentialsEnc = encrypted
	}

	reg.UpdatedAt = time.Now()

	if err := h.repo.Registries.Update(c.Context(), reg); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update registry")
	}

	if input.IsDefault != nil && *input.IsDefault {
		_ = h.repo.Registries.SetDefault(c.Context(), reg.ProjectID, reg.ID)
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "registry", registryID, input)

	return c.JSON(reg)
}

// DeleteRegistry deletes a registry.
func (h *Handler) DeleteRegistry(c fiber.Ctx) error {
	registryID := c.Params("rid")

	if _, err := h.repo.Registries.GetByID(c.Context(), registryID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	if err := h.repo.Registries.Delete(c.Context(), registryID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete registry")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "registry", registryID, nil)

	return c.JSON(fiber.Map{"message": "registry deleted"})
}

// TestRegistry validates registry credentials.
func (h *Handler) TestRegistry(c fiber.Ctx) error {
	registryID := c.Params("rid")

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	// Decrypt credentials
	password, err := h.decryptRegistryCredentials(reg)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "message": "failed to decrypt credentials: " + err.Error()})
	}

	client, err := registry.NewClient(reg, password)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "message": "failed to create registry client: " + err.Error()})
	}

	if err := client.ValidateCredentials(c.Context()); err != nil {
		return c.JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "connection successful"})
}

// ListRegistryImages lists images in a registry.
func (h *Handler) ListRegistryImages(c fiber.Ctx) error {
	registryID := c.Params("rid")

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	password, err := h.decryptRegistryCredentials(reg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decrypt credentials")
	}

	client, err := registry.NewClient(reg, password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create registry client")
	}

	limit := 50
	images, err := client.ListImages(c.Context(), limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list images: "+err.Error())
	}

	return c.JSON(images)
}

// ListRegistryTags lists tags for a specific image in a registry.
func (h *Handler) ListRegistryTags(c fiber.Ctx) error {
	registryID := c.Params("rid")
	imageName := c.Params("name")

	// URL-decode the image name (it may contain slashes encoded as %2F)
	decoded, err := url.PathUnescape(imageName)
	if err == nil {
		imageName = decoded
	}

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	password, err := h.decryptRegistryCredentials(reg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decrypt credentials")
	}

	client, err := registry.NewClient(reg, password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create registry client")
	}

	tags, err := client.ListTags(c.Context(), imageName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tags: "+err.Error())
	}

	return c.JSON(tags)
}

// DeleteRegistryTag deletes a specific tag from a registry.
func (h *Handler) DeleteRegistryTag(c fiber.Ctx) error {
	registryID := c.Params("rid")
	imageName := c.Params("name")
	tag := c.Params("tag")

	// URL-decode the image name
	decoded, err := url.PathUnescape(imageName)
	if err == nil {
		imageName = decoded
	}

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	password, err := h.decryptRegistryCredentials(reg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decrypt credentials")
	}

	client, err := registry.NewClient(reg, password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create registry client")
	}

	if err := client.DeleteTag(c.Context(), imageName, tag); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete tag: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete_tag", "registry", registryID,
		fiber.Map{"image": imageName, "tag": tag})

	return c.JSON(fiber.Map{"message": "tag deleted"})
}

// SetDefaultRegistry sets a registry as the default for a project.
func (h *Handler) SetDefaultRegistry(c fiber.Ctx) error {
	projectID := c.Params("id")
	registryID := c.Params("rid")

	reg, err := h.repo.Registries.GetByID(c.Context(), registryID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not found")
	}

	if reg.ProjectID != projectID {
		return fiber.NewError(fiber.StatusBadRequest, "registry does not belong to this project")
	}

	if err := h.repo.Registries.SetDefault(c.Context(), projectID, registryID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to set default registry")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "set_default", "registry", registryID,
		fiber.Map{"project_id": projectID})

	return c.JSON(fiber.Map{"message": "default registry updated"})
}

// decryptRegistryCredentials decrypts the stored credentials for a registry.
func (h *Handler) decryptRegistryCredentials(reg *models.Registry) (string, error) {
	if reg.CredentialsEnc == "" {
		return "", nil
	}
	encKey, err := hex.DecodeString(h.cfg.EncryptionKey)
	if err != nil {
		return "", err
	}
	return crypto.Decrypt(encKey, reg.CredentialsEnc)
}
