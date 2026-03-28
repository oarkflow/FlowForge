package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/oarkflow/deploy/backend/internal/detector"
	"github.com/oarkflow/deploy/backend/internal/importer"
	"github.com/oarkflow/deploy/backend/internal/integrations"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/pipeline"
	"github.com/oarkflow/deploy/backend/pkg/crypto"
)

// --------------------------------------------------------------------------
// POST /api/v1/import/detect
// --------------------------------------------------------------------------

type importDetectRequest struct {
	SourceType string `json:"source_type"` // git, github, gitlab, bitbucket, local, upload
	GitURL     string `json:"git_url,omitempty"`
	SSHKey     string `json:"ssh_key,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Provider   string `json:"provider,omitempty"`
	RepoOwner  string `json:"repo_owner,omitempty"`
	RepoName   string `json:"repo_name,omitempty"`
	LocalPath  string `json:"local_path,omitempty"`
	UploadID   string `json:"upload_id,omitempty"`
}

type importDetectResponse struct {
	SessionID         string                       `json:"session_id"`
	Detections        []detector.DetectionResult   `json:"detections"`
	Profile           detector.ProjectProfile      `json:"profile"`
	Repository        importer.RepositoryMetadata  `json:"repository"`
	GeneratedPipeline string                       `json:"generated_pipeline"`
	DefaultBranch     string                       `json:"default_branch"`
	CloneURL          string                       `json:"clone_url"`
	ExtractedEnvVars  []pipeline.ExtractedVariable `json:"extracted_env_vars"`
	ExtractedSecrets  []pipeline.ExtractedVariable `json:"extracted_secrets"`
}

func (h *Handler) ImportDetect(c fiber.Ctx) error {
	if h.importer == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "config",
			"message": "Import service not configured",
		})
	}

	var req importDetectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "Invalid request body",
		})
	}

	if req.SourceType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "source_type is required",
		})
	}

	// For upload source type, resolve the upload path from session store.
	uploadPath := ""
	if req.SourceType == "upload" && req.UploadID != "" {
		uploadPath = h.importer.Sessions().Get(req.UploadID)
		if uploadPath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation",
				"message": "Upload not found or expired. Please upload again.",
			})
		}
	}

	// Get provider token from header for OAuth providers.
	providerToken := c.Get("X-Provider-Token")

	importReq := importer.ImportRequest{
		SourceType:  req.SourceType,
		GitURL:      req.GitURL,
		SSHKeyPEM:   req.SSHKey,
		Branch:      req.Branch,
		Provider:    req.Provider,
		RepoOwner:   req.RepoOwner,
		RepoName:    req.RepoName,
		AccessToken: providerToken,
		LocalPath:   req.LocalPath,
		UploadPath:  uploadPath,
	}

	result, err := h.importer.Import(c.Context(), importReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "import",
			"message": fmt.Sprintf("Import failed: %v", err),
		})
	}

	// Store the work dir in session for later use in project creation.
	sessionID := h.importer.Sessions().Create(result.WorkDir)

	// Extract env vars and secrets referenced in the pipeline YAML.
	extraction := pipeline.ExtractVariables(result.GeneratedYAML)

	return c.Status(fiber.StatusOK).JSON(importDetectResponse{
		SessionID:         sessionID,
		Detections:        result.Detections,
		Profile:           result.Profile,
		Repository:        result.Repository,
		GeneratedPipeline: result.GeneratedYAML,
		DefaultBranch:     result.DefaultBranch,
		CloneURL:          result.CloneURL,
		ExtractedEnvVars:  extraction.EnvVars,
		ExtractedSecrets:  extraction.Secrets,
	})
}

// --------------------------------------------------------------------------
// POST /api/v1/import/upload
// --------------------------------------------------------------------------

func (h *Handler) ImportUpload(c fiber.Ctx) error {
	if h.importer == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "config",
			"message": "Import service not configured",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "No file uploaded. Use multipart field 'file'.",
		})
	}

	// Validate file extension.
	filename := file.Filename
	lower := strings.ToLower(filename)
	if !strings.HasSuffix(lower, ".zip") && !strings.HasSuffix(lower, ".tar.gz") && !strings.HasSuffix(lower, ".tgz") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "Unsupported file format. Accepted: .zip, .tar.gz, .tgz",
		})
	}

	// Save uploaded file to temp location.
	tmpDir, err := os.MkdirTemp("", "flowforge-upload-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upload",
			"message": "Failed to create temporary directory",
		})
	}

	archivePath := filepath.Join(tmpDir, filename)
	if err := c.SaveFile(file, archivePath); err != nil {
		os.RemoveAll(tmpDir)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upload",
			"message": "Failed to save uploaded file",
		})
	}

	// Extract archive.
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "upload",
			"message": "Failed to create extraction directory",
		})
	}

	if err := importer.Extract(archivePath, extractDir); err != nil {
		os.RemoveAll(tmpDir)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "extraction",
			"message": fmt.Sprintf("Failed to extract archive: %v", err),
		})
	}

	// Unwrap single-subfolder archives: if the extraction produced a single
	// directory entry inside extractDir (e.g. extracted/my-project/), use that
	// inner directory as the working directory so the detector sees the project
	// root directly rather than a wrapper folder.
	extractDir = importer.UnwrapSingleSubfolder(extractDir)

	// Store in session so /detect can find it.
	uploadID := h.importer.Sessions().Create(extractDir)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"upload_id": uploadID,
		"filename":  filename,
	})
}

// --------------------------------------------------------------------------
// GET /api/v1/import/providers/:provider/repos
// --------------------------------------------------------------------------

func (h *Handler) ImportListRepos(c fiber.Ctx) error {
	if h.importer == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "config",
			"message": "Import service not configured",
		})
	}

	provider := c.Params("provider")
	if provider != "github" && provider != "gitlab" && provider != "bitbucket" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "Unsupported provider. Use github, gitlab, or bitbucket.",
		})
	}

	token := c.Get("X-Provider-Token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "auth",
			"message": "X-Provider-Token header is required",
		})
	}

	page := 1
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	perPage := 20
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			perPage = n
		}
	}

	opts := integrations.ListReposOptions{
		Search:  c.Query("search"),
		Page:    page,
		PerPage: perPage,
	}

	repos, total, err := h.importer.ListProviderRepos(c.Context(), provider, token, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "provider",
			"message": fmt.Sprintf("Failed to list repos: %v", err),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"repos": repos,
		"total": total,
		"page":  page,
	})
}

// --------------------------------------------------------------------------
// POST /api/v1/import/project
// --------------------------------------------------------------------------

type importProjectRequest struct {
	SessionID        string                       `json:"session_id"`
	Project          importProjectData            `json:"project"`
	Repository       importRepoData               `json:"repository"`
	PipelineYAML     string                       `json:"pipeline_yaml"`
	SetupWebhook     bool                         `json:"setup_webhook"`
	ExtractedEnvVars []pipeline.ExtractedVariable `json:"extracted_env_vars,omitempty"`
	ExtractedSecrets []pipeline.ExtractedVariable `json:"extracted_secrets,omitempty"`
}

type importProjectData struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	OrgID       string `json:"org_id,omitempty"`
}

type importRepoData struct {
	Provider      string `json:"provider"`
	ProviderID    string `json:"provider_id"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url,omitempty"`
	DefaultBranch string `json:"default_branch"`
	AccessToken   string `json:"access_token,omitempty"`
}

func (h *Handler) ImportCreateProject(c fiber.Ctx) error {
	if h.importer == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "config",
			"message": "Import service not configured",
		})
	}

	var req importProjectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "Invalid request body",
		})
	}

	if req.Project.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "validation",
			"message": "Project name is required",
		})
	}

	// Generate slug if not provided.
	slug := req.Project.Slug
	if slug == "" {
		slug = toSlug(req.Project.Name)
	}

	visibility := req.Project.Visibility
	if visibility == "" {
		visibility = "private"
	}

	userID := getUserID(c)
	projectID := uuid.New().String()

	// Begin transaction.
	tx, err := h.db.Beginx()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database",
			"message": "Failed to start transaction",
		})
	}
	defer tx.Rollback()

	// 1. Create project.
	orgID := interface{}(nil)
	if req.Project.OrgID != "" {
		orgID = req.Project.OrgID
	}
	_, err = tx.Exec(`INSERT INTO projects (id, org_id, name, slug, description, visibility, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		projectID, orgID, req.Project.Name, slug, req.Project.Description, visibility, userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database",
			"message": fmt.Sprintf("Failed to create project: %v", err),
		})
	}

	// 2. Create repository (if clone URL provided).
	var repoID string
	if req.Repository.CloneURL != "" || req.Repository.FullName != "" {
		repoID = uuid.New().String()
		provider := req.Repository.Provider
		if provider == "" {
			provider = "git"
		}
		providerID := req.Repository.ProviderID
		if providerID == "" {
			providerID = req.Repository.FullName
		}
		fullName := req.Repository.FullName
		if fullName == "" {
			fullName = req.Repository.CloneURL
		}
		defaultBranch := req.Repository.DefaultBranch
		if defaultBranch == "" {
			defaultBranch = "main"
		}

		// Encrypt access token if provided.
		var accessTokenEnc interface{}
		providerToken := c.Get("X-Provider-Token")
		if providerToken == "" {
			providerToken = req.Repository.AccessToken
		}
		if providerToken != "" {
			encKey, _ := getEncKey(h.cfg.EncryptionKey)
			if len(encKey) == 32 {
				enc, err := crypto.Encrypt(encKey, providerToken)
				if err == nil {
					accessTokenEnc = enc
				}
			}
		}

		_, err = tx.Exec(`INSERT INTO repositories (id, project_id, provider, provider_id, full_name, clone_url, ssh_url, default_branch, access_token_enc)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			repoID, projectID, provider, providerID, fullName, req.Repository.CloneURL,
			nilIfEmpty(req.Repository.SSHURL), defaultBranch, accessTokenEnc,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "database",
				"message": fmt.Sprintf("Failed to create repository: %v", err),
			})
		}
	}

	// 3. Create pipeline (if YAML provided).
	var pipelineID string
	if req.PipelineYAML != "" {
		pipelineID = uuid.New().String()
		pipelineName := req.Project.Name + " CI"

		var repoIDVal interface{}
		if repoID != "" {
			repoIDVal = repoID
		}

		_, err = tx.Exec(`INSERT INTO pipelines (id, project_id, repository_id, name, config_source, config_content, config_version, triggers, created_by)
			VALUES (?, ?, ?, ?, 'db', ?, 1, '{}', ?)`,
			pipelineID, projectID, repoIDVal, pipelineName, req.PipelineYAML, userID,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "database",
				"message": fmt.Sprintf("Failed to create pipeline: %v", err),
			})
		}

		// Create initial pipeline version.
		versionID := uuid.New().String()
		_, err = tx.Exec(`INSERT INTO pipeline_versions (id, pipeline_id, version, config, message, created_by)
			VALUES (?, ?, 1, ?, 'Initial auto-generated pipeline', ?)`,
			versionID, pipelineID, req.PipelineYAML, userID,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "database",
				"message": fmt.Sprintf("Failed to create pipeline version: %v", err),
			})
		}
	}

	// Commit transaction.
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database",
			"message": "Failed to commit transaction",
		})
	}

	// Clean up import session.
	if req.SessionID != "" {
		h.importer.Sessions().Remove(req.SessionID)
	}

	// Audit log.
	h.audit.LogAction(c.Context(), userID, getClientIP(c), "import_project", "project", projectID, nil)

	// Auto-create extracted env vars with empty values so they appear
	// in the project settings, ready for the user to fill in.
	for _, ev := range req.ExtractedEnvVars {
		if ev.HasValue {
			continue
		}
		_ = h.repo.EnvVars.Create(c.Context(), &models.EnvVar{
			ProjectID: projectID,
			Key:       ev.Name,
			Value:     "",
		})
	}

	// Auto-create extracted secrets with a placeholder empty value.
	// SecretStore.Create rejects empty strings, so we use the repo
	// directly with an encrypted empty value.
	for _, sec := range req.ExtractedSecrets {
		_ = h.secretStore.CreateEmpty(c.Context(), projectID, sec.Name, userID)
	}

	// Fetch created resources for response.
	var project models.Project
	_ = h.db.Get(&project, "SELECT * FROM projects WHERE id = ?", projectID)

	var repo *models.Repository
	if repoID != "" {
		var r models.Repository
		if err := h.db.Get(&r, "SELECT * FROM repositories WHERE id = ?", repoID); err == nil {
			repo = &r
		}
	}

	var pipeline *models.Pipeline
	if pipelineID != "" {
		var p models.Pipeline
		if err := h.db.Get(&p, "SELECT * FROM pipelines WHERE id = ?", pipelineID); err == nil {
			pipeline = &p
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"project":    project,
		"repository": repo,
		"pipeline":   pipeline,
	})
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func getEncKey(hexKey string) ([]byte, error) {
	if hexKey == "" {
		return nil, fmt.Errorf("no encryption key")
	}
	key := make([]byte, 32)
	n := 0
	for i := 0; i < len(hexKey) && n < 32; i += 2 {
		if i+2 > len(hexKey) {
			break
		}
		b, err := strconv.ParseUint(hexKey[i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		key[n] = byte(b)
		n++
	}
	if n != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}
	return key, nil
}
