package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"

	"github.com/oarkflow/deploy/backend/internal/api/handlers"
	"github.com/oarkflow/deploy/backend/internal/api/middleware"
	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/importer"
)

func RegisterRoutes(app *fiber.App, db *sqlx.DB, cfg *config.Config, imp *importer.Service) {
	h := handlers.New(db, cfg, imp)

	api := app.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
	auth.Post("/refresh", h.RefreshToken)
	auth.Post("/logout", h.Logout)

	// System (health is public, metrics/info require admin)
	system := api.Group("/system")
	system.Get("/health", h.HealthCheck)
	system.Get("/metrics", middleware.Auth(cfg.JWTSecret), middleware.RequireAdmin(), h.Metrics)
	system.Get("/info", middleware.Auth(cfg.JWTSecret), middleware.RequireAdmin(), h.SystemInfo)

	// Protected routes
	protected := api.Group("", middleware.Auth(cfg.JWTSecret))

	// Global runs (all pipelines/projects)
	protected.Get("/runs", h.ListAllRuns)

	// Import wizard
	importGroup := protected.Group("/import")
	importGroup.Post("/detect", h.ImportDetect)
	importGroup.Post("/upload", h.ImportUpload)
	importGroup.Get("/providers/:provider/repos", h.ImportListRepos)
	importGroup.Post("/project", h.ImportCreateProject)

	// Users
	users := protected.Group("/users")
	users.Get("/me", h.GetCurrentUser)
	users.Put("/me", h.UpdateCurrentUser)
	users.Get("/:id", h.GetUser)

	// Organizations
	orgs := protected.Group("/orgs")
	orgs.Get("/", h.ListOrgs)
	orgs.Post("/", middleware.RequireAdmin(), h.CreateOrg)
	orgs.Get("/:id", h.GetOrg)
	orgs.Put("/:id", middleware.RequireAdmin(), h.UpdateOrg)
	orgs.Delete("/:id", middleware.RequireOwner(), h.DeleteOrg)
	orgs.Get("/:id/members", h.ListOrgMembers)
	orgs.Post("/:id/members", middleware.RequireAdmin(), h.AddOrgMember)
	orgs.Delete("/:id/members/:userId", middleware.RequireAdmin(), h.RemoveOrgMember)

	// Projects
	projects := protected.Group("/projects")
	projects.Get("/", h.ListProjects)
	projects.Post("/", middleware.RequireDev(), h.CreateProject)
	projects.Get("/:id", h.GetProject)
	projects.Put("/:id", middleware.RequireDev(), h.UpdateProject)
	projects.Delete("/:id", middleware.RequireAdmin(), h.DeleteProject)

	// Repositories
	projects.Get("/:id/repositories", h.ListRepositories)
	projects.Post("/:id/repositories", middleware.RequireDev(), h.CreateRepository)
	projects.Delete("/:id/repositories/:repoId", middleware.RequireAdmin(), h.DeleteRepository)
	projects.Post("/:id/repositories/:repoId/sync", middleware.RequireDev(), h.SyncRepository)

	// Pipelines
	pipelines := projects.Group("/:id/pipelines")
	pipelines.Get("/", h.ListPipelines)
	pipelines.Post("/", middleware.RequireDev(), h.CreatePipeline)
	pipelines.Get("/:pid", h.GetPipeline)
	pipelines.Put("/:pid", middleware.RequireDev(), h.UpdatePipeline)
	pipelines.Delete("/:pid", middleware.RequireAdmin(), h.DeletePipeline)
	pipelines.Get("/:pid/versions", h.ListPipelineVersions)
	pipelines.Post("/:pid/trigger", middleware.RequireDev(), h.TriggerPipeline)
	pipelines.Post("/:pid/validate", middleware.RequireDev(), h.ValidatePipeline)

	// Pipeline Runs
	runs := pipelines.Group("/:pid/runs")
	runs.Get("/", h.ListRuns)
	runs.Get("/:rid", h.GetRun)
	runs.Post("/:rid/cancel", middleware.RequireDev(), h.CancelRun)
	runs.Post("/:rid/rerun", middleware.RequireDev(), h.RerunPipeline)
	runs.Post("/:rid/approve", middleware.RequireAdmin(), h.ApproveRun)
	runs.Get("/:rid/logs", h.GetRunLogs)
	runs.Get("/:rid/artifacts", h.GetRunArtifacts)

	// Secrets
	secrets := projects.Group("/:id/secrets")
	secrets.Get("/", middleware.RequireDev(), h.ListSecrets)
	secrets.Post("/", middleware.RequireAdmin(), h.CreateSecret)
	secrets.Put("/:secretId", middleware.RequireAdmin(), h.UpdateSecret)
	secrets.Delete("/:secretId", middleware.RequireAdmin(), h.DeleteSecret)

	// Notifications
	notifs := projects.Group("/:id/notifications")
	notifs.Get("/", h.ListNotificationChannels)
	notifs.Post("/", middleware.RequireAdmin(), h.CreateNotificationChannel)
	notifs.Put("/:nid", middleware.RequireAdmin(), h.UpdateNotificationChannel)
	notifs.Delete("/:nid", middleware.RequireAdmin(), h.DeleteNotificationChannel)

	// Agents
	agents := protected.Group("/agents")
	agents.Get("/", h.ListAgents)
	agents.Post("/", middleware.RequireAdmin(), h.CreateAgent)
	agents.Get("/:id", h.GetAgent)
	agents.Delete("/:id", middleware.RequireAdmin(), h.DeleteAgent)
	agents.Post("/:id/drain", middleware.RequireAdmin(), h.DrainAgent)

	// Artifacts
	artifacts := protected.Group("/artifacts")
	artifacts.Get("/:id", h.GetArtifact)
	artifacts.Get("/:id/download", h.DownloadArtifact)

	// Audit logs
	protected.Get("/audit-logs", middleware.RequireAdmin(), h.ListAuditLogs)

	// Webhooks (unauthenticated, signature-validated)
	webhooks := app.Group("/webhooks")
	webhooks.Post("/github", h.GithubWebhook)
	webhooks.Post("/gitlab", h.GitlabWebhook)
	webhooks.Post("/bitbucket", h.BitbucketWebhook)
}
