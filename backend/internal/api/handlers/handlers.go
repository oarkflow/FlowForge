package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/auth"
	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/importer"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/pipeline"
	"github.com/oarkflow/deploy/backend/internal/secrets"
)

// PipelineEngine is the interface the handler uses to trigger pipeline runs.
// This avoids a direct dependency on the engine package.
type PipelineEngine interface {
	TriggerPipeline(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) (*models.PipelineRun, error)
}

type Handler struct {
	db          *sqlx.DB
	cfg         *config.Config
	repo        *queries.Repositories
	secretStore *secrets.SecretStore
	audit       *auth.AuditLogger
	importer    *importer.Service
	engine      PipelineEngine
}

func New(db *sqlx.DB, cfg *config.Config, imp *importer.Service, engine PipelineEngine) *Handler {
	repos := queries.NewRepositories(db)
	encKey, _ := hex.DecodeString(cfg.EncryptionKey)
	return &Handler{
		db:          db,
		cfg:         cfg,
		repo:        repos,
		secretStore: secrets.NewSecretStore(repos, encKey),
		audit:       auth.NewAuditLogger(repos.AuditLogs),
		importer:    imp,
		engine:      engine,
	}
}

// --------------------------------------------------------------------------
// Helper: get pagination params from query string
// --------------------------------------------------------------------------

func (h *Handler) pagination(c fiber.Ctx) (limit, offset int) {
	limit = 100
	offset = 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return
}

// --------------------------------------------------------------------------
// Helper: generate slug from name
// --------------------------------------------------------------------------

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "unnamed"
	}
	return slug
}

// --------------------------------------------------------------------------
// Helper: get user ID from locals (set by auth middleware)
// --------------------------------------------------------------------------

func getUserID(c fiber.Ctx) string {
	uid, _ := c.Locals("user_id").(string)
	return uid
}

func getClientIP(c fiber.Ctx) string {
	return c.IP()
}

// =========================================================================
// AUTH HANDLERS
// =========================================================================

type loginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *Handler) Login(c fiber.Ctx) error {
	var input loginInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	user, err := h.repo.Users.GetByEmail(c.Context(), input.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	if user.IsActive == 0 {
		return fiber.NewError(fiber.StatusForbidden, "account is deactivated")
	}
	if user.PasswordHash == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	if !auth.CheckPassword(*user.PasswordHash, input.Password) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, user.Email, user.Username, user.Role,
		h.cfg.JWTSecret, h.cfg.JWTExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate access token")
	}

	refreshToken, err := auth.GenerateRefreshToken(
		user.ID, h.cfg.JWTSecret, h.cfg.RefreshExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	_ = h.audit.LogAction(c.Context(), user.ID, getClientIP(c), "login", "user", user.ID, nil)

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.JWTExpiration.Seconds()),
		"user":          user,
	})
}

type registerInput struct {
	Email       string `json:"email" validate:"required,email"`
	Username    string `json:"username" validate:"required,min=3,max=50"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name"`
}

func (h *Handler) Register(c fiber.Ctx) error {
	var input registerInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" || input.Username == "" || input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email, username, and password are required")
	}
	if len(input.Password) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	// Check for existing email
	if existing, _ := h.repo.Users.GetByEmail(c.Context(), input.Email); existing != nil {
		return fiber.NewError(fiber.StatusConflict, "email is already registered")
	}
	// Check for existing username
	if existing, _ := h.repo.Users.GetByUsername(c.Context(), input.Username); existing != nil {
		return fiber.NewError(fiber.StatusConflict, "username is already taken")
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	user := &models.User{
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: &hashedPassword,
		Role:         "developer",
		IsActive:     1,
	}
	if input.DisplayName != "" {
		user.DisplayName = &input.DisplayName
	}

	if err := h.repo.Users.Create(c.Context(), user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create user: "+err.Error())
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, user.Email, user.Username, user.Role,
		h.cfg.JWTSecret, h.cfg.JWTExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate access token")
	}

	refreshToken, err := auth.GenerateRefreshToken(
		user.ID, h.cfg.JWTSecret, h.cfg.RefreshExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	_ = h.audit.LogAction(c.Context(), user.ID, getClientIP(c), "register", "user", user.ID, nil)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.JWTExpiration.Seconds()),
		"user":          user,
	})
}

type refreshInput struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *Handler) RefreshToken(c fiber.Ctx) error {
	var input refreshInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	claims, err := auth.ParseRefreshToken(input.RefreshToken, h.cfg.JWTSecret)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	user, err := h.repo.Users.GetByID(c.Context(), claims.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}
	if user.IsActive == 0 {
		return fiber.NewError(fiber.StatusForbidden, "account is deactivated")
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, user.Email, user.Username, user.Role,
		h.cfg.JWTSecret, h.cfg.JWTExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate access token")
	}

	newRefreshToken, err := auth.GenerateRefreshToken(
		user.ID, h.cfg.JWTSecret, h.cfg.RefreshExpiration,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(h.cfg.JWTExpiration.Seconds()),
	})
}

func (h *Handler) Logout(c fiber.Ctx) error {
	userID := getUserID(c)
	if userID != "" {
		_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "logout", "user", userID, nil)
	}
	return c.JSON(fiber.Map{"message": "logged out successfully"})
}

// =========================================================================
// USER HANDLERS
// =========================================================================

func (h *Handler) GetCurrentUser(c fiber.Ctx) error {
	userID := getUserID(c)
	user, err := h.repo.Users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(user)
}

type updateUserInput struct {
	Email       *string `json:"email" validate:"omitempty,email"`
	Username    *string `json:"username" validate:"omitempty,min=3,max=50"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

func (h *Handler) UpdateCurrentUser(c fiber.Ctx) error {
	userID := getUserID(c)
	var input updateUserInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	user, err := h.repo.Users.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	if input.Email != nil && *input.Email != "" {
		// Check uniqueness
		existing, _ := h.repo.Users.GetByEmail(c.Context(), *input.Email)
		if existing != nil && existing.ID != userID {
			return fiber.NewError(fiber.StatusConflict, "email is already in use")
		}
		user.Email = *input.Email
	}
	if input.Username != nil && *input.Username != "" {
		existing, _ := h.repo.Users.GetByUsername(c.Context(), *input.Username)
		if existing != nil && existing.ID != userID {
			return fiber.NewError(fiber.StatusConflict, "username is already taken")
		}
		user.Username = *input.Username
	}
	if input.DisplayName != nil {
		user.DisplayName = input.DisplayName
	}
	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}

	if err := h.repo.Users.Update(c.Context(), user); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update user")
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "update", "user", userID, input)

	return c.JSON(user)
}

func (h *Handler) GetUser(c fiber.Ctx) error {
	user, err := h.repo.Users.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(user)
}

// =========================================================================
// ORGANIZATION HANDLERS
// =========================================================================

func (h *Handler) ListOrgs(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	orgs, err := h.repo.Orgs.List(c.Context(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list organizations")
	}
	return c.JSON(orgs)
}

type createOrgInput struct {
	Name    string  `json:"name" validate:"required"`
	Slug    string  `json:"slug"`
	LogoURL *string `json:"logo_url"`
}

func (h *Handler) CreateOrg(c fiber.Ctx) error {
	var input createOrgInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	slug := input.Slug
	if slug == "" {
		slug = toSlug(input.Name)
	}

	org := &models.Organization{
		Name:    input.Name,
		Slug:    slug,
		LogoURL: input.LogoURL,
	}

	if err := h.repo.Orgs.Create(c.Context(), org); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create organization: "+err.Error())
	}

	// Add creator as owner member
	userID := getUserID(c)
	_ = h.repo.Orgs.AddMember(c.Context(), &models.OrgMember{
		OrgID:  org.ID,
		UserID: userID,
		Role:   "owner",
	})

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "organization", org.ID, input)

	return c.Status(fiber.StatusCreated).JSON(org)
}

func (h *Handler) GetOrg(c fiber.Ctx) error {
	org, err := h.repo.Orgs.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}
	return c.JSON(org)
}

type updateOrgInput struct {
	Name    *string `json:"name"`
	Slug    *string `json:"slug"`
	LogoURL *string `json:"logo_url"`
}

func (h *Handler) UpdateOrg(c fiber.Ctx) error {
	orgID := c.Params("id")
	var input updateOrgInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	org, err := h.repo.Orgs.GetByID(c.Context(), orgID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}

	if input.Name != nil && *input.Name != "" {
		org.Name = *input.Name
	}
	if input.Slug != nil && *input.Slug != "" {
		org.Slug = *input.Slug
	}
	if input.LogoURL != nil {
		org.LogoURL = input.LogoURL
	}

	if err := h.repo.Orgs.Update(c.Context(), org); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update organization")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "organization", orgID, input)

	return c.JSON(org)
}

func (h *Handler) DeleteOrg(c fiber.Ctx) error {
	orgID := c.Params("id")
	if _, err := h.repo.Orgs.GetByID(c.Context(), orgID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}

	if err := h.repo.Orgs.Delete(c.Context(), orgID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete organization")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "organization", orgID, nil)

	return c.JSON(fiber.Map{"message": "organization deleted"})
}

func (h *Handler) ListOrgMembers(c fiber.Ctx) error {
	members, err := h.repo.Orgs.ListMembers(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list members")
	}
	return c.JSON(members)
}

type addMemberInput struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=owner admin developer viewer"`
}

func (h *Handler) AddOrgMember(c fiber.Ctx) error {
	orgID := c.Params("id")
	var input addMemberInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.UserID == "" || input.Role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "user_id and role are required")
	}

	// Verify org exists
	if _, err := h.repo.Orgs.GetByID(c.Context(), orgID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "organization not found")
	}
	// Verify user exists
	if _, err := h.repo.Users.GetByID(c.Context(), input.UserID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: input.UserID,
		Role:   input.Role,
	}
	if err := h.repo.Orgs.AddMember(c.Context(), member); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to add member: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "add_member", "organization", orgID, input)

	return c.Status(fiber.StatusCreated).JSON(member)
}

func (h *Handler) RemoveOrgMember(c fiber.Ctx) error {
	orgID := c.Params("id")
	memberUserID := c.Params("userId")

	if err := h.repo.Orgs.RemoveMember(c.Context(), orgID, memberUserID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to remove member")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "remove_member", "organization", orgID,
		fiber.Map{"removed_user_id": memberUserID})

	return c.JSON(fiber.Map{"message": "member removed"})
}

// =========================================================================
// PROJECT HANDLERS
// =========================================================================

func (h *Handler) ListProjects(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	orgID := c.Query("org_id")
	var projects []models.Project
	var err error
	if orgID != "" {
		projects, err = h.repo.Projects.ListByOrg(c.Context(), orgID, limit, offset)
	} else {
		projects, err = h.repo.Projects.List(c.Context(), limit, offset)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list projects")
	}
	return c.JSON(projects)
}

type createProjectInput struct {
	Name        string  `json:"name" validate:"required"`
	Slug        string  `json:"slug"`
	OrgID       *string `json:"org_id"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility" validate:"omitempty,oneof=private internal public"`
}

func (h *Handler) CreateProject(c fiber.Ctx) error {
	var input createProjectInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	slug := input.Slug
	if slug == "" {
		slug = toSlug(input.Name)
	}
	visibility := input.Visibility
	if visibility == "" {
		visibility = "private"
	}

	userID := getUserID(c)
	project := &models.Project{
		OrgID:       input.OrgID,
		Name:        input.Name,
		Slug:        slug,
		Description: input.Description,
		Visibility:  visibility,
		CreatedBy:   &userID,
	}

	if err := h.repo.Projects.Create(c.Context(), project); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create project: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "project", project.ID, input)

	return c.Status(fiber.StatusCreated).JSON(project)
}

func (h *Handler) GetProject(c fiber.Ctx) error {
	project, err := h.repo.Projects.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	return c.JSON(project)
}

type updateProjectInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility" validate:"omitempty,oneof=private internal public"`
}

func (h *Handler) UpdateProject(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input updateProjectInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	project, err := h.repo.Projects.GetByID(c.Context(), projectID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	if input.Name != nil && *input.Name != "" {
		project.Name = *input.Name
	}
	if input.Slug != nil && *input.Slug != "" {
		project.Slug = *input.Slug
	}
	if input.Description != nil {
		project.Description = input.Description
	}
	if input.Visibility != nil && *input.Visibility != "" {
		project.Visibility = *input.Visibility
	}

	if err := h.repo.Projects.Update(c.Context(), project); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update project")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "project", projectID, input)

	return c.JSON(project)
}

func (h *Handler) DeleteProject(c fiber.Ctx) error {
	projectID := c.Params("id")
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	if err := h.repo.Projects.SoftDelete(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete project")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "project", projectID, nil)

	return c.JSON(fiber.Map{"message": "project deleted"})
}

// =========================================================================
// REPOSITORY HANDLERS
// =========================================================================

func (h *Handler) ListRepositories(c fiber.Ctx) error {
	repos, err := h.repo.Repos.ListByProjectID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list repositories")
	}
	return c.JSON(repos)
}

type createRepoInput struct {
	Provider      string  `json:"provider" validate:"required,oneof=github gitlab bitbucket"`
	ProviderID    string  `json:"provider_id" validate:"required"`
	FullName      string  `json:"full_name" validate:"required"`
	CloneURL      string  `json:"clone_url" validate:"required"`
	SSHURL        *string `json:"ssh_url"`
	DefaultBranch string  `json:"default_branch"`
}

func (h *Handler) CreateRepository(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input createRepoInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Provider == "" || input.FullName == "" || input.CloneURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "provider, full_name, and clone_url are required")
	}

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	defaultBranch := input.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	repo := &models.Repository{
		ProjectID:     projectID,
		Provider:      input.Provider,
		ProviderID:    input.ProviderID,
		FullName:      input.FullName,
		CloneURL:      input.CloneURL,
		SSHURL:        input.SSHURL,
		DefaultBranch: defaultBranch,
		IsActive:      1,
	}

	if err := h.repo.Repos.Create(c.Context(), repo); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create repository: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "repository", repo.ID, input)

	return c.Status(fiber.StatusCreated).JSON(repo)
}

func (h *Handler) DeleteRepository(c fiber.Ctx) error {
	repoID := c.Params("repoId")
	if _, err := h.repo.Repos.GetByID(c.Context(), repoID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "repository not found")
	}

	if err := h.repo.Repos.Delete(c.Context(), repoID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete repository")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "repository", repoID, nil)

	return c.JSON(fiber.Map{"message": "repository deleted"})
}

func (h *Handler) SyncRepository(c fiber.Ctx) error {
	repoID := c.Params("repoId")
	repo, err := h.repo.Repos.GetByID(c.Context(), repoID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "repository not found")
	}

	// Mark last sync time — actual sync logic would be handled by the engine
	_ = repo
	return c.JSON(fiber.Map{
		"message":     "sync initiated",
		"repository":  repo.FullName,
		"last_sync_at": time.Now(),
	})
}

// =========================================================================
// PIPELINE HANDLERS
// =========================================================================

func (h *Handler) ListPipelines(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	pipelines, err := h.repo.Pipelines.ListByProject(c.Context(), c.Params("id"), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list pipelines")
	}
	return c.JSON(pipelines)
}

type createPipelineInput struct {
	Name          string  `json:"name" validate:"required"`
	Description   *string `json:"description"`
	RepositoryID  *string `json:"repository_id"`
	ConfigSource  string  `json:"config_source" validate:"omitempty,oneof=db repo"`
	ConfigPath    *string `json:"config_path"`
	ConfigContent *string `json:"config_content"`
	Triggers      *string `json:"triggers"`
}

func (h *Handler) CreatePipeline(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input createPipelineInput
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

	configSource := input.ConfigSource
	if configSource == "" {
		configSource = "db"
	}

	// If config content is provided and source is "db", parse and validate it
	if input.ConfigContent != nil && *input.ConfigContent != "" && configSource == "db" {
		_, validationErrs, err := pipeline.ParseAndValidate(*input.ConfigContent)
		if err != nil && validationErrs == nil {
			// Parse error
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "pipeline config parse error",
				"message": err.Error(),
			})
		}
		if len(validationErrs) > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":            "pipeline config validation failed",
				"validation_errors": validationErrs,
			})
		}
	}

	triggers := "{}"
	if input.Triggers != nil {
		triggers = *input.Triggers
	}

	configPath := ".flowforge.yml"
	if input.ConfigPath != nil && *input.ConfigPath != "" {
		configPath = *input.ConfigPath
	}

	userID := getUserID(c)
	p := &models.Pipeline{
		ProjectID:     projectID,
		RepositoryID:  input.RepositoryID,
		Name:          input.Name,
		Description:   input.Description,
		ConfigSource:  configSource,
		ConfigPath:    &configPath,
		ConfigContent: input.ConfigContent,
		ConfigVersion: 1,
		Triggers:      triggers,
		IsActive:      1,
		CreatedBy:     &userID,
	}

	if err := h.repo.Pipelines.Create(c.Context(), p); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create pipeline: "+err.Error())
	}

	// Create initial version if config content was provided
	if input.ConfigContent != nil && *input.ConfigContent != "" {
		msg := "Initial version"
		_ = h.repo.Pipelines.CreateVersion(c.Context(), &models.PipelineVersion{
			PipelineID: p.ID,
			Version:    1,
			Config:     *input.ConfigContent,
			Message:    &msg,
			CreatedBy:  &userID,
		})
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "pipeline", p.ID,
		fiber.Map{"name": input.Name, "config_source": configSource})

	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *Handler) GetPipeline(c fiber.Ctx) error {
	p, err := h.repo.Pipelines.GetByID(c.Context(), c.Params("pid"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}
	return c.JSON(p)
}

type updatePipelineInput struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	ConfigContent *string `json:"config_content"`
	ConfigPath    *string `json:"config_path"`
	Triggers      *string `json:"triggers"`
	IsActive      *int    `json:"is_active"`
	Message       *string `json:"message"`
}

func (h *Handler) UpdatePipeline(c fiber.Ctx) error {
	pipelineID := c.Params("pid")
	var input updatePipelineInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	p, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}

	if input.Name != nil && *input.Name != "" {
		p.Name = *input.Name
	}
	if input.Description != nil {
		p.Description = input.Description
	}
	if input.ConfigPath != nil {
		p.ConfigPath = input.ConfigPath
	}
	if input.Triggers != nil {
		p.Triggers = *input.Triggers
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}

	// If config content changed, parse/validate and bump version
	if input.ConfigContent != nil && *input.ConfigContent != "" {
		if p.ConfigSource == "db" {
			_, validationErrs, parseErr := pipeline.ParseAndValidate(*input.ConfigContent)
			if parseErr != nil && validationErrs == nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "pipeline config parse error",
					"message": parseErr.Error(),
				})
			}
			if len(validationErrs) > 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":            "pipeline config validation failed",
					"validation_errors": validationErrs,
				})
			}
		}

		p.ConfigContent = input.ConfigContent
		p.ConfigVersion++

		// Create a new version record
		userID := getUserID(c)
		_ = h.repo.Pipelines.CreateVersion(c.Context(), &models.PipelineVersion{
			PipelineID: p.ID,
			Version:    p.ConfigVersion,
			Config:     *input.ConfigContent,
			Message:    input.Message,
			CreatedBy:  &userID,
		})
	}

	if err := h.repo.Pipelines.Update(c.Context(), p); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update pipeline")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "pipeline", pipelineID, input)

	return c.JSON(p)
}

func (h *Handler) DeletePipeline(c fiber.Ctx) error {
	pipelineID := c.Params("pid")
	if _, err := h.repo.Pipelines.GetByID(c.Context(), pipelineID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "pipeline not found")
	}

	if err := h.repo.Pipelines.SoftDelete(c.Context(), pipelineID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete pipeline")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "pipeline", pipelineID, nil)

	return c.JSON(fiber.Map{"message": "pipeline deleted"})
}

func (h *Handler) ListPipelineVersions(c fiber.Ctx) error {
	pipelineID := c.Params("pid")
	versions, err := h.repo.Pipelines.ListVersions(c.Context(), pipelineID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list pipeline versions")
	}
	return c.JSON(versions)
}

type triggerPipelineInput struct {
	Branch  *string            `json:"branch"`
	Tag     *string            `json:"tag"`
	Inputs  map[string]string  `json:"inputs"`
}

func (h *Handler) TriggerPipeline(c fiber.Ctx) error {
	pipelineID := c.Params("pid")
	var input triggerPipelineInput
	// Input is optional for manual triggers
	_ = c.Bind().JSON(&input)

	// Build trigger data map for the engine
	triggerData := make(map[string]string)
	if input.Branch != nil {
		triggerData["branch"] = *input.Branch
	}
	if input.Tag != nil {
		triggerData["tag"] = *input.Tag
	}
	for k, v := range input.Inputs {
		triggerData[k] = v
	}

	userID := getUserID(c)
	triggerData["created_by"] = userID

	run, err := h.engine.TriggerPipeline(c.Context(), pipelineID, "manual", triggerData)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to trigger pipeline: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "trigger", "pipeline", pipelineID,
		fiber.Map{"run_id": run.ID, "run_number": run.Number})

	return c.Status(fiber.StatusCreated).JSON(run)
}

type validatePipelineInput struct {
	Config string `json:"config" validate:"required"`
}

func (h *Handler) ValidatePipeline(c fiber.Ctx) error {
	var input validatePipelineInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Config == "" {
		return fiber.NewError(fiber.StatusBadRequest, "config is required")
	}

	spec, validationErrs, err := pipeline.ParseAndValidate(input.Config)
	if err != nil && validationErrs == nil {
		// Pure parse error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"valid":   false,
			"error":   "parse error",
			"message": err.Error(),
		})
	}

	if len(validationErrs) > 0 {
		return c.JSON(fiber.Map{
			"valid":            false,
			"validation_errors": validationErrs,
		})
	}

	// Return summary of parsed spec
	jobNames := make([]string, 0, len(spec.Jobs))
	for name := range spec.Jobs {
		jobNames = append(jobNames, name)
	}

	return c.JSON(fiber.Map{
		"valid":   true,
		"name":    spec.Name,
		"version": spec.Version,
		"stages":  spec.Stages,
		"jobs":    jobNames,
	})
}

// =========================================================================
// RUN HANDLERS
// =========================================================================

func (h *Handler) ListAllRuns(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	status := c.Query("status")
	runs, err := h.repo.Runs.ListAllRuns(c.Context(), status, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list runs")
	}
	return c.JSON(runs)
}

func (h *Handler) ListRuns(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	runs, err := h.repo.Runs.ListByPipeline(c.Context(), c.Params("pid"), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list runs")
	}
	return c.JSON(runs)
}

func (h *Handler) GetRun(c fiber.Ctx) error {
	run, err := h.repo.Runs.GetByID(c.Context(), c.Params("rid"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	// Enrich with full stage→job→step hierarchy
	stages, _ := h.repo.Runs.ListStageRuns(c.Context(), run.ID)
	jobs, _ := h.repo.Runs.ListJobRunsByRunID(c.Context(), run.ID)
	steps, _ := h.repo.Runs.ListStepRunsByRunID(c.Context(), run.ID)

	type enrichedRun struct {
		*models.PipelineRun
		Stages []models.StageRun `json:"stages,omitempty"`
		Jobs   []models.JobRun   `json:"jobs,omitempty"`
		Steps  []models.StepRun  `json:"steps,omitempty"`
	}

	return c.JSON(enrichedRun{
		PipelineRun: run,
		Stages:      stages,
		Jobs:        jobs,
		Steps:       steps,
	})
}

func (h *Handler) CancelRun(c fiber.Ctx) error {
	runID := c.Params("rid")
	run, err := h.repo.Runs.GetByID(c.Context(), runID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	// Can only cancel queued or running runs
	if run.Status != "queued" && run.Status != "running" && run.Status != "pending" {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("cannot cancel run with status %q", run.Status))
	}

	if err := h.repo.Runs.UpdateStatus(c.Context(), runID, "cancelled"); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to cancel run")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "cancel", "pipeline_run", runID, nil)

	return c.JSON(fiber.Map{"message": "run cancelled", "run_id": runID})
}

func (h *Handler) RerunPipeline(c fiber.Ctx) error {
	runID := c.Params("rid")
	origRun, err := h.repo.Runs.GetByID(c.Context(), runID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	userID := getUserID(c)

	// Build trigger data from the original run
	triggerData := map[string]string{
		"rerun_of":   runID,
		"created_by": userID,
	}
	if origRun.Branch != nil {
		triggerData["branch"] = *origRun.Branch
	}
	if origRun.CommitSHA != nil {
		triggerData["commit_sha"] = *origRun.CommitSHA
	}
	if origRun.CommitMessage != nil {
		triggerData["commit_message"] = *origRun.CommitMessage
	}
	if origRun.Author != nil {
		triggerData["author"] = *origRun.Author
	}
	if origRun.Tag != nil {
		triggerData["tag"] = *origRun.Tag
	}

	newRun, err := h.engine.TriggerPipeline(c.Context(), origRun.PipelineID, origRun.TriggerType, triggerData)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to rerun pipeline: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "rerun", "pipeline_run", runID,
		fiber.Map{"new_run_id": newRun.ID})

	return c.Status(fiber.StatusCreated).JSON(newRun)
}

func (h *Handler) ApproveRun(c fiber.Ctx) error {
	runID := c.Params("rid")
	run, err := h.repo.Runs.GetByID(c.Context(), runID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	if run.Status != "waiting_approval" {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("run is not waiting for approval (current status: %s)", run.Status))
	}

	// Re-trigger through the engine so the run gets enqueued
	triggerData := make(map[string]string)
	if run.Branch != nil {
		triggerData["branch"] = *run.Branch
	}
	if run.CommitSHA != nil {
		triggerData["commit_sha"] = *run.CommitSHA
	}
	if run.CommitMessage != nil {
		triggerData["commit_message"] = *run.CommitMessage
	}
	if run.Author != nil {
		triggerData["author"] = *run.Author
	}

	newRun, err := h.engine.TriggerPipeline(c.Context(), run.PipelineID, run.TriggerType, triggerData)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to approve run: "+err.Error())
	}

	// Mark the original waiting run as cancelled since we created a new one
	_ = h.repo.Runs.UpdateStatus(c.Context(), runID, "cancelled")

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "approve", "pipeline_run", runID, nil)

	return c.JSON(fiber.Map{"message": "run approved", "run_id": newRun.ID, "run_number": newRun.Number})
}

func (h *Handler) GetRunLogs(c fiber.Ctx) error {
	runID := c.Params("rid")

	// Verify run exists
	if _, err := h.repo.Runs.GetByID(c.Context(), runID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	limit, offset := h.pagination(c)
	// Allow larger limit for logs
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 10000 {
			limit = l
		}
	} else {
		limit = 1000
	}

	// Optional filter by step
	stepRunID := c.Query("step_run_id")

	var logs []models.RunLog
	var err error
	if stepRunID != "" {
		logs, err = h.repo.Logs.GetByStepRunID(c.Context(), stepRunID, limit, offset)
	} else {
		logs, err = h.repo.Logs.GetByRunID(c.Context(), runID, limit, offset)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch logs")
	}

	return c.JSON(logs)
}

func (h *Handler) GetRunArtifacts(c fiber.Ctx) error {
	runID := c.Params("rid")
	// Verify run exists
	if _, err := h.repo.Runs.GetByID(c.Context(), runID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}

	artifacts, err := h.repo.Artifacts.ListByRunID(c.Context(), runID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list artifacts")
	}
	return c.JSON(artifacts)
}

// =========================================================================
// SECRET HANDLERS
// =========================================================================

func (h *Handler) ListSecrets(c fiber.Ctx) error {
	projectID := c.Params("id")
	limit, offset := h.pagination(c)

	// Use the SecretStore.List which returns metadata only (no decrypted values)
	secretMeta, err := h.secretStore.List(c.Context(), projectID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list secrets")
	}
	return c.JSON(secretMeta)
}

type createSecretInput struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}

func (h *Handler) CreateSecret(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input createSecretInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Key == "" || input.Value == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key and value are required")
	}

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	userID := getUserID(c)
	if err := h.secretStore.Create(c.Context(), projectID, input.Key, input.Value, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create secret: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), userID, getClientIP(c), "create", "secret", "",
		fiber.Map{"project_id": projectID, "key": input.Key})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "secret created",
		"key":        input.Key,
		"project_id": projectID,
	})
}

type updateSecretInput struct {
	Value string `json:"value" validate:"required"`
}

func (h *Handler) UpdateSecret(c fiber.Ctx) error {
	secretID := c.Params("secretId")
	var input updateSecretInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Value == "" {
		return fiber.NewError(fiber.StatusBadRequest, "value is required")
	}

	if err := h.secretStore.Update(c.Context(), secretID, input.Value); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update secret: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "secret", secretID, nil)

	return c.JSON(fiber.Map{"message": "secret updated"})
}

func (h *Handler) DeleteSecret(c fiber.Ctx) error {
	secretID := c.Params("secretId")
	if err := h.secretStore.Delete(c.Context(), secretID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete secret: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "secret", secretID, nil)

	return c.JSON(fiber.Map{"message": "secret deleted"})
}

// =========================================================================
// NOTIFICATION HANDLERS
// =========================================================================

func (h *Handler) ListNotificationChannels(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	channels, err := h.repo.Notifications.ListByProject(c.Context(), c.Params("id"), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list notification channels")
	}
	return c.JSON(channels)
}

type createNotifChannelInput struct {
	Type     string `json:"type" validate:"required,oneof=slack email teams discord pagerduty webhook"`
	Name     string `json:"name" validate:"required"`
	Config   string `json:"config" validate:"required"` // JSON config to be encrypted
	IsActive *int   `json:"is_active"`
}

func (h *Handler) CreateNotificationChannel(c fiber.Ctx) error {
	projectID := c.Params("id")
	var input createNotifChannelInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Type == "" || input.Name == "" || input.Config == "" {
		return fiber.NewError(fiber.StatusBadRequest, "type, name, and config are required")
	}

	// Verify project exists
	if _, err := h.repo.Projects.GetByID(c.Context(), projectID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	isActive := 1
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	channel := &models.NotificationChannel{
		ProjectID: &projectID,
		Type:      input.Type,
		Name:      input.Name,
		ConfigEnc: input.Config, // In production, encrypt this with crypto.Encrypt
		IsActive:  isActive,
	}

	if err := h.repo.Notifications.Create(c.Context(), channel); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create notification channel")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "notification_channel", channel.ID,
		fiber.Map{"type": input.Type, "name": input.Name})

	return c.Status(fiber.StatusCreated).JSON(channel)
}

type updateNotifChannelInput struct {
	Name     *string `json:"name"`
	Config   *string `json:"config"`
	IsActive *int    `json:"is_active"`
}

func (h *Handler) UpdateNotificationChannel(c fiber.Ctx) error {
	channelID := c.Params("nid")
	var input updateNotifChannelInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	channel, err := h.repo.Notifications.GetByID(c.Context(), channelID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "notification channel not found")
	}

	if input.Name != nil && *input.Name != "" {
		channel.Name = *input.Name
	}
	if input.Config != nil && *input.Config != "" {
		channel.ConfigEnc = *input.Config
	}
	if input.IsActive != nil {
		channel.IsActive = *input.IsActive
	}

	if err := h.repo.Notifications.Update(c.Context(), channel); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update notification channel")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "update", "notification_channel", channelID, input)

	return c.JSON(channel)
}

func (h *Handler) DeleteNotificationChannel(c fiber.Ctx) error {
	channelID := c.Params("nid")
	if _, err := h.repo.Notifications.GetByID(c.Context(), channelID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "notification channel not found")
	}

	if err := h.repo.Notifications.Delete(c.Context(), channelID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete notification channel")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "notification_channel", channelID, nil)

	return c.JSON(fiber.Map{"message": "notification channel deleted"})
}

// =========================================================================
// AGENT HANDLERS
// =========================================================================

func (h *Handler) ListAgents(c fiber.Ctx) error {
	limit, offset := h.pagination(c)
	statusFilter := c.Query("status")
	var agents []models.Agent
	var err error
	if statusFilter != "" {
		agents, err = h.repo.Agents.ListByStatus(c.Context(), statusFilter)
	} else {
		agents, err = h.repo.Agents.List(c.Context(), limit, offset)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list agents")
	}
	return c.JSON(agents)
}

type createAgentInput struct {
	Name     string   `json:"name" validate:"required"`
	Labels   []string `json:"labels"`
	Executor string   `json:"executor" validate:"omitempty,oneof=local docker kubernetes"`
}

func (h *Handler) CreateAgent(c fiber.Ctx) error {
	var input createAgentInput
	if err := c.Bind().JSON(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	executor := input.Executor
	if executor == "" {
		executor = "local"
	}

	// Generate an API key for the agent
	plainKey, hashedKey, err := auth.GenerateAPIKey()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate agent token")
	}

	labelsJSON, _ := json.Marshal(input.Labels)

	agent := &models.Agent{
		Name:      input.Name,
		TokenHash: hashedKey,
		Labels:    string(labelsJSON),
		Executor:  executor,
		Status:    "offline",
	}

	if err := h.repo.Agents.Create(c.Context(), agent); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create agent: "+err.Error())
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "create", "agent", agent.ID,
		fiber.Map{"name": input.Name, "executor": executor})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"agent": agent,
		"token": plainKey, // Show token only once
	})
}

func (h *Handler) GetAgent(c fiber.Ctx) error {
	agent, err := h.repo.Agents.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "agent not found")
	}
	return c.JSON(agent)
}

func (h *Handler) DeleteAgent(c fiber.Ctx) error {
	agentID := c.Params("id")
	agent, err := h.repo.Agents.GetByID(c.Context(), agentID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "agent not found")
	}

	if agent.Status == "busy" {
		return fiber.NewError(fiber.StatusConflict, "cannot delete agent while it is busy; drain it first")
	}

	if err := h.repo.Agents.Delete(c.Context(), agentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete agent")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "delete", "agent", agentID, nil)

	return c.JSON(fiber.Map{"message": "agent deleted"})
}

func (h *Handler) DrainAgent(c fiber.Ctx) error {
	agentID := c.Params("id")
	if _, err := h.repo.Agents.GetByID(c.Context(), agentID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "agent not found")
	}

	if err := h.repo.Agents.UpdateStatus(c.Context(), agentID, "draining"); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to drain agent")
	}

	_ = h.audit.LogAction(c.Context(), getUserID(c), getClientIP(c), "drain", "agent", agentID, nil)

	return c.JSON(fiber.Map{"message": "agent draining", "agent_id": agentID})
}

// =========================================================================
// ARTIFACT HANDLERS
// =========================================================================

func (h *Handler) GetArtifact(c fiber.Ctx) error {
	artifact, err := h.repo.Artifacts.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "artifact not found")
	}
	return c.JSON(artifact)
}

func (h *Handler) DownloadArtifact(c fiber.Ctx) error {
	artifact, err := h.repo.Artifacts.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "artifact not found")
	}

	// For local storage, serve the file directly
	if artifact.StorageBackend == "local" {
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, artifact.Name))
		return c.SendFile(artifact.StorageKey)
	}

	// For remote storage (S3, etc.), return a redirect or presigned URL
	// This is a placeholder — in production, generate a presigned URL
	return c.JSON(fiber.Map{
		"message":         "download not available for this storage backend",
		"storage_backend": artifact.StorageBackend,
		"storage_key":     artifact.StorageKey,
	})
}

// =========================================================================
// AUDIT LOG HANDLER
// =========================================================================

func (h *Handler) ListAuditLogs(c fiber.Ctx) error {
	limit, offset := h.pagination(c)

	// Support filtering by actor or resource
	actorID := c.Query("actor_id")
	resource := c.Query("resource")
	resourceID := c.Query("resource_id")

	var logs []models.AuditLog
	var err error

	if actorID != "" {
		logs, err = h.repo.AuditLogs.ListByActor(c.Context(), actorID, limit, offset)
	} else if resource != "" && resourceID != "" {
		logs, err = h.repo.AuditLogs.ListByResource(c.Context(), resource, resourceID, limit, offset)
	} else {
		logs, err = h.repo.AuditLogs.List(c.Context(), limit, offset)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit logs")
	}
	return c.JSON(logs)
}

// =========================================================================
// SYSTEM HANDLERS
// =========================================================================

func (h *Handler) HealthCheck(c fiber.Ctx) error {
	// Verify database is reachable
	if err := h.db.PingContext(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "unhealthy",
			"error":   "database unreachable",
		})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) Metrics(c fiber.Ctx) error {
	// Gather basic metrics
	agents, _ := h.repo.Agents.List(c.Context(), 10000, 0)
	onlineAgents := 0
	busyAgents := 0
	for _, a := range agents {
		switch a.Status {
		case "online":
			onlineAgents++
		case "busy":
			busyAgents++
		}
	}

	return c.JSON(fiber.Map{
		"agents_total":  len(agents),
		"agents_online": onlineAgents,
		"agents_busy":   busyAgents,
		"timestamp":     time.Now(),
	})
}

func (h *Handler) SystemInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": "1.0.0",
		"name":    "FlowForge",
		"go":      "1.26.1",
	})
}

// =========================================================================
// WEBHOOK HANDLERS
// =========================================================================

type githubWebhookEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		ID       int    `json:"id"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
	PullRequest *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		Head   struct {
			SHA string `json:"sha"`
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request,omitempty"`
}

func (h *Handler) GithubWebhook(c fiber.Ctx) error {
	body := c.Body()

	// Validate HMAC-SHA256 signature
	sigHeader := c.Get("X-Hub-Signature-256")
	eventType := c.Get("X-GitHub-Event")

	if eventType == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing X-GitHub-Event header")
	}

	// Try to find the matching repository by provider ID to get the webhook secret
	var event githubWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON payload")
	}

	repoFullName := event.Repository.FullName
	if repoFullName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing repository information")
	}

	// Look up the repository in our DB to get the webhook secret
	providerID := strconv.Itoa(event.Repository.ID)
	var matchedRepos []models.Repository
	// Search by listing all repos (in production, add a GetByProviderID query)
	allProjects, _ := h.repo.Projects.List(c.Context(), 10000, 0)
	for _, proj := range allProjects {
		repos, _ := h.repo.Repos.ListByProjectID(c.Context(), proj.ID)
		for _, r := range repos {
			if r.Provider == "github" && (r.ProviderID == providerID || r.FullName == repoFullName) {
				matchedRepos = append(matchedRepos, r)
			}
		}
	}

	if len(matchedRepos) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "no matching repository found")
	}

	// Validate signature for each matching repo
	var validRepo *models.Repository
	for i, repo := range matchedRepos {
		if repo.WebhookSecret != nil && *repo.WebhookSecret != "" {
			if validateGitHubSignature(body, *repo.WebhookSecret, sigHeader) {
				validRepo = &matchedRepos[i]
				break
			}
		} else {
			// No secret configured, accept the webhook (not recommended in production)
			validRepo = &matchedRepos[i]
			break
		}
	}

	if validRepo == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid webhook signature")
	}

	// Determine trigger type and create pipeline runs
	var triggerType string
	var branch, commitSHA, commitMsg, author string

	switch eventType {
	case "push":
		triggerType = "push"
		// Extract branch from ref (refs/heads/main → main)
		branch = strings.TrimPrefix(event.Ref, "refs/heads/")
		branch = strings.TrimPrefix(branch, "refs/tags/")
		commitSHA = event.After
		if event.HeadCommit.ID != "" {
			commitSHA = event.HeadCommit.ID
			commitMsg = event.HeadCommit.Message
			author = event.HeadCommit.Author.Name
		}

	case "pull_request":
		triggerType = "pull_request"
		if event.PullRequest != nil {
			branch = event.PullRequest.Head.Ref
			commitSHA = event.PullRequest.Head.SHA
			commitMsg = event.PullRequest.Title
		}

	case "ping":
		return c.JSON(fiber.Map{"message": "pong"})

	default:
		// Acknowledge but don't process
		return c.JSON(fiber.Map{"message": "event received", "event": eventType})
	}

	// Find pipelines for this repository and create runs
	pipelines, _ := h.repo.Pipelines.ListByProject(c.Context(), validRepo.ProjectID, 1000, 0)

	var createdRuns []fiber.Map
	for _, p := range pipelines {
		if p.IsActive == 0 {
			continue
		}
		if p.RepositoryID == nil || *p.RepositoryID != validRepo.ID {
			continue
		}

		triggerData := map[string]string{
			"event":      eventType,
			"repository": repoFullName,
			"sender":     author,
		}
		if branch != "" {
			triggerData["branch"] = branch
		}
		if commitSHA != "" {
			triggerData["commit_sha"] = commitSHA
		}
		if commitMsg != "" {
			triggerData["commit_message"] = commitMsg
		}
		if author != "" {
			triggerData["author"] = author
		}

		run, err := h.engine.TriggerPipeline(c.Context(), p.ID, triggerType, triggerData)
		if err != nil {
			continue
		}

		createdRuns = append(createdRuns, fiber.Map{
			"pipeline_id": p.ID,
			"run_id":      run.ID,
			"run_number":  run.Number,
		})
	}

	return c.JSON(fiber.Map{
		"message":      "webhook processed",
		"event":        eventType,
		"repository":   repoFullName,
		"runs_created": len(createdRuns),
		"runs":         createdRuns,
	})
}

func (h *Handler) GitlabWebhook(c fiber.Ctx) error {
	body := c.Body()
	gitlabToken := c.Get("X-Gitlab-Token")

	// Parse the event to identify the repository
	var payload struct {
		EventName  string `json:"event_name"`
		ObjectKind string `json:"object_kind"`
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Project    struct {
			ID                int    `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Commits []struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commits"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON payload")
	}

	// Find matching repository
	providerID := strconv.Itoa(payload.Project.ID)
	var matchedRepo *models.Repository
	allProjects, _ := h.repo.Projects.List(c.Context(), 10000, 0)
	for _, proj := range allProjects {
		repos, _ := h.repo.Repos.ListByProjectID(c.Context(), proj.ID)
		for i, r := range repos {
			if r.Provider == "gitlab" && (r.ProviderID == providerID || r.FullName == payload.Project.PathWithNamespace) {
				matchedRepo = &repos[i]
				break
			}
		}
		if matchedRepo != nil {
			break
		}
	}

	if matchedRepo == nil {
		return fiber.NewError(fiber.StatusNotFound, "no matching repository found")
	}

	// Validate GitLab token
	if matchedRepo.WebhookSecret != nil && *matchedRepo.WebhookSecret != "" {
		if gitlabToken != *matchedRepo.WebhookSecret {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid webhook token")
		}
	}

	// Determine trigger type
	triggerType := "push"
	if payload.ObjectKind == "merge_request" {
		triggerType = "pull_request"
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	var commitSHA, commitMsg, author string
	if len(payload.Commits) > 0 {
		lastCommit := payload.Commits[len(payload.Commits)-1]
		commitSHA = lastCommit.ID
		commitMsg = lastCommit.Message
		author = lastCommit.Author.Name
	} else {
		commitSHA = payload.After
	}

	// Create pipeline runs
	pipelines, _ := h.repo.Pipelines.ListByProject(c.Context(), matchedRepo.ProjectID, 1000, 0)
	runsCreated := 0
	for _, p := range pipelines {
		if p.IsActive == 0 || p.RepositoryID == nil || *p.RepositoryID != matchedRepo.ID {
			continue
		}

		triggerData := map[string]string{
			"event":      payload.ObjectKind,
			"repository": payload.Project.PathWithNamespace,
		}
		if branch != "" {
			triggerData["branch"] = branch
		}
		if commitSHA != "" {
			triggerData["commit_sha"] = commitSHA
		}
		if commitMsg != "" {
			triggerData["commit_message"] = commitMsg
		}
		if author != "" {
			triggerData["author"] = author
		}

		if _, err := h.engine.TriggerPipeline(c.Context(), p.ID, triggerType, triggerData); err == nil {
			runsCreated++
		}
	}

	return c.JSON(fiber.Map{
		"message":      "webhook processed",
		"event":        payload.ObjectKind,
		"runs_created": runsCreated,
	})
}

func (h *Handler) BitbucketWebhook(c fiber.Ctx) error {
	body := c.Body()
	eventType := c.Get("X-Event-Key")

	var payload struct {
		Repository struct {
			UUID     string `json:"uuid"`
			FullName string `json:"full_name"`
		} `json:"repository"`
		Push *struct {
			Changes []struct {
				New struct {
					Name   string `json:"name"`
					Target struct {
						Hash    string `json:"hash"`
						Message string `json:"message"`
						Author  struct {
							Raw string `json:"raw"`
						} `json:"author"`
					} `json:"target"`
				} `json:"new"`
			} `json:"changes"`
		} `json:"push"`
		PullRequest *struct {
			Title  string `json:"title"`
			Source struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
				Commit struct {
					Hash string `json:"hash"`
				} `json:"commit"`
			} `json:"source"`
		} `json:"pullrequest"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON payload")
	}

	// Find matching repository
	var matchedRepo *models.Repository
	allProjects, _ := h.repo.Projects.List(c.Context(), 10000, 0)
	for _, proj := range allProjects {
		repos, _ := h.repo.Repos.ListByProjectID(c.Context(), proj.ID)
		for i, r := range repos {
			if r.Provider == "bitbucket" && (r.ProviderID == payload.Repository.UUID || r.FullName == payload.Repository.FullName) {
				matchedRepo = &repos[i]
				break
			}
		}
		if matchedRepo != nil {
			break
		}
	}

	if matchedRepo == nil {
		return fiber.NewError(fiber.StatusNotFound, "no matching repository found")
	}

	var triggerType, branch, commitSHA, commitMsg, author string

	switch {
	case strings.HasPrefix(eventType, "repo:push"):
		triggerType = "push"
		if payload.Push != nil && len(payload.Push.Changes) > 0 {
			change := payload.Push.Changes[0]
			branch = change.New.Name
			commitSHA = change.New.Target.Hash
			commitMsg = change.New.Target.Message
			author = change.New.Target.Author.Raw
		}
	case strings.HasPrefix(eventType, "pullrequest:"):
		triggerType = "pull_request"
		if payload.PullRequest != nil {
			branch = payload.PullRequest.Source.Branch.Name
			commitSHA = payload.PullRequest.Source.Commit.Hash
			commitMsg = payload.PullRequest.Title
		}
	default:
		return c.JSON(fiber.Map{"message": "event received", "event": eventType})
	}

	pipelines, _ := h.repo.Pipelines.ListByProject(c.Context(), matchedRepo.ProjectID, 1000, 0)
	runsCreated := 0
	for _, p := range pipelines {
		if p.IsActive == 0 || p.RepositoryID == nil || *p.RepositoryID != matchedRepo.ID {
			continue
		}

		triggerData := map[string]string{
			"event":      eventType,
			"repository": payload.Repository.FullName,
		}
		if branch != "" {
			triggerData["branch"] = branch
		}
		if commitSHA != "" {
			triggerData["commit_sha"] = commitSHA
		}
		if commitMsg != "" {
			triggerData["commit_message"] = commitMsg
		}
		if author != "" {
			triggerData["author"] = author
		}

		if _, err := h.engine.TriggerPipeline(c.Context(), p.ID, triggerType, triggerData); err == nil {
			runsCreated++
		}
	}

	return c.JSON(fiber.Map{
		"message":      "webhook processed",
		"event":        eventType,
		"runs_created": runsCreated,
	})
}

// =========================================================================
// HELPER FUNCTIONS
// =========================================================================

// validateGitHubSignature validates the HMAC-SHA256 signature from GitHub webhooks.
func validateGitHubSignature(payload []byte, secret, sigHeader string) bool {
	if sigHeader == "" {
		return false
	}
	sigHex := strings.TrimPrefix(sigHeader, "sha256=")
	if sigHex == sigHeader {
		return false // missing sha256= prefix
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigHex), []byte(expectedMAC))
}

// strPtrOrNil returns a pointer to s if non-empty, nil otherwise.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
