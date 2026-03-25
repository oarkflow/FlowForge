package api

import (
	"bytes"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"

	agentpkg "github.com/oarkflow/deploy/backend/internal/agent"
	"github.com/oarkflow/deploy/backend/internal/api/handlers"
	"github.com/oarkflow/deploy/backend/internal/api/middleware"
	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/importer"
)

func RegisterRoutes(app *fiber.App, db *sqlx.DB, cfg *config.Config, imp *importer.Service, agentServer *agentpkg.Server, engine handlers.PipelineEngine) {
	h := handlers.New(db, cfg, imp, engine)

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
	projects.Put("/:id/repositories/:repoId", middleware.RequireDev(), h.UpdateRepository)
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

	// Environment Variables
	envVars := projects.Group("/:id/env-vars")
	envVars.Get("/", middleware.RequireDev(), h.ListEnvVars)
	envVars.Post("/", middleware.RequireDev(), h.CreateEnvVar)
	envVars.Put("/", middleware.RequireDev(), h.BulkSaveEnvVars) // Bulk upsert
	envVars.Put("/:varId", middleware.RequireDev(), h.UpdateEnvVar)
	envVars.Delete("/:varId", middleware.RequireDev(), h.DeleteEnvVar)

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

	// Agent communication endpoints (unauthenticated — token validated by agent server)
	agentComm := api.Group("/agents")
	if agentServer != nil {
		agentComm.Post("/register", wrapHTTPHandler(agentServer.HandleRegister))
		agentComm.Post("/heartbeat", wrapHTTPHandler(agentServer.HandleHeartbeat))
		agentComm.Post("/poll", wrapHTTPHandler(agentServer.HandleJobPoll))
		agentComm.Post("/complete", wrapHTTPHandler(agentServer.HandleJobComplete))
		agentComm.Post("/log", wrapHTTPHandler(agentServer.HandleLog))
	}
}

// wrapHTTPHandler adapts a net/http handler function to a Fiber handler.
func wrapHTTPHandler(handler func(w http.ResponseWriter, r *http.Request)) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Use Fiber's built-in adaptor: read body, build http.Request, capture response
		c.Set("Content-Type", "application/json")

		// Create a response writer adapter
		w := &fiberResponseWriter{ctx: c, statusCode: 200}
		r, err := http.NewRequest(c.Method(), c.OriginalURL(), bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to create request"})
		}
		r.Header.Set("Content-Type", "application/json")

		handler(w, r)

		if w.written {
			return nil
		}
		return c.Status(w.statusCode).Send(w.body)
	}
}

// fiberResponseWriter adapts Fiber's context to net/http.ResponseWriter.
type fiberResponseWriter struct {
	ctx        fiber.Ctx
	statusCode int
	headers    http.Header
	body       []byte
	written    bool
}

func (w *fiberResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *fiberResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *fiberResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}
