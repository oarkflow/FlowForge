package api

import (
	"bytes"
	"context"
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

	// Recent deployments (for dashboard)
	protected.Get("/deployments/recent", h.ListRecentDeployments)

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
	secrets.Put("/:secretId/rotation", middleware.RequireAdmin(), h.SetSecretRotationPolicy)
	secrets.Post("/:secretId/rotate", middleware.RequireAdmin(), h.MarkSecretRotated)
	secrets.Get("/rotation-status", middleware.RequireDev(), h.ListRotationStatus)

	// Secret Providers (external: Vault, AWS, GCP)
	secretProviders := projects.Group("/:id/secret-providers")
	secretProviders.Get("/", middleware.RequireDev(), h.ListSecretProviders)
	secretProviders.Post("/", middleware.RequireAdmin(), h.CreateSecretProvider)
	secretProviders.Put("/:spid", middleware.RequireAdmin(), h.UpdateSecretProvider)
	secretProviders.Delete("/:spid", middleware.RequireAdmin(), h.DeleteSecretProvider)
	secretProviders.Get("/health", middleware.RequireDev(), h.GetSecretProviderHealth)

	// Project IP Allowlist
	projectIPAllowlist := projects.Group("/:id/ip-allowlist")
	projectIPAllowlist.Get("/", middleware.RequireAdmin(), h.ListIPAllowlist)
	projectIPAllowlist.Post("/", middleware.RequireAdmin(), h.CreateIPAllowlistEntry)
	projectIPAllowlist.Delete("/:entryId", middleware.RequireAdmin(), h.DeleteIPAllowlistEntry)

	// Project Deployment Providers
	deploymentProviders := projects.Group("/:id/deployment-providers")
	deploymentProviders.Get("/", middleware.RequireDev(), h.ListDeploymentProviders)
	deploymentProviders.Post("/", middleware.RequireAdmin(), h.CreateDeploymentProvider)
	deploymentProviders.Put("/:dpid", middleware.RequireAdmin(), h.UpdateDeploymentProvider)
	deploymentProviders.Delete("/:dpid", middleware.RequireAdmin(), h.DeleteDeploymentProvider)
	deploymentProviders.Post("/:dpid/test", middleware.RequireDev(), h.TestDeploymentProvider)

	// Project Environment Chain
	environmentChain := projects.Group("/:id/environment-chain")
	environmentChain.Get("/", middleware.RequireDev(), h.GetEnvironmentChain)
	environmentChain.Put("/", middleware.RequireDev(), h.UpdateEnvironmentChain)

	// Pipeline Stage -> Environment mappings
	projects.Get("/:id/pipelines/:pid/stage-environments", middleware.RequireDev(), h.GetPipelineStageEnvironmentMappings)
	projects.Put("/:id/pipelines/:pid/stage-environments", middleware.RequireDev(), h.UpdatePipelineStageEnvironmentMappings)

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

	// Environments
	environments := projects.Group("/:id/environments")
	environments.Get("/", h.ListEnvironments)
	environments.Post("/", middleware.RequireDev(), h.CreateEnvironment)
	environments.Put("/:eid", middleware.RequireDev(), h.UpdateEnvironment)
	environments.Delete("/:eid", middleware.RequireAdmin(), h.DeleteEnvironment)
	environments.Post("/:eid/lock", middleware.RequireDev(), h.LockEnvironment)
	environments.Post("/:eid/unlock", middleware.RequireDev(), h.UnlockEnvironment)

	// Deployments
	environments.Get("/:eid/deployments", h.ListDeployments)
	environments.Post("/:eid/deploy", middleware.RequireDev(), h.TriggerDeployment)
	environments.Post("/:eid/rollback", middleware.RequireDev(), h.RollbackDeployment)
	environments.Post("/:eid/promote", middleware.RequireDev(), h.PromoteDeployment)

	// Deployment Strategy
	environments.Get("/:eid/strategy", middleware.RequireDev(), h.GetStrategyConfig)
	environments.Put("/:eid/strategy", middleware.RequireDev(), h.UpdateStrategyConfig)
	environments.Post("/:eid/deployments/:did/advance-canary", middleware.RequireDev(), h.AdvanceCanary)
	environments.Get("/:eid/deployments/:did/health", h.CheckDeploymentHealth)
	environments.Get("/:eid/deployments/:did/plan", h.GetDeploymentPlan)

	// Environment Overrides
	environments.Get("/:eid/overrides", middleware.RequireDev(), h.ListEnvOverrides)
	environments.Put("/:eid/overrides", middleware.RequireDev(), h.SaveEnvOverrides)

	// Environment Protection Rules
	environments.Put("/:eid/protection", middleware.RequireAdmin(), h.UpdateProtectionRules)

	// Registries
	registries := projects.Group("/:id/registries")
	registries.Get("/", h.ListRegistries)
	registries.Post("/", middleware.RequireDev(), h.CreateRegistry)
	registries.Put("/:rid", middleware.RequireDev(), h.UpdateRegistry)
	registries.Delete("/:rid", middleware.RequireAdmin(), h.DeleteRegistry)
	registries.Post("/:rid/test", middleware.RequireDev(), h.TestRegistry)
	registries.Get("/:rid/images", h.ListRegistryImages)
	registries.Get("/:rid/images/:name/tags", h.ListRegistryTags)
	registries.Delete("/:rid/images/:name/tags/:tag", middleware.RequireDev(), h.DeleteRegistryTag)
	registries.Post("/:rid/default", middleware.RequireDev(), h.SetDefaultRegistry)

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

	// Approvals
	approvals := protected.Group("/approvals")
	approvals.Get("/pending", h.ListPendingApprovals)
	approvals.Get("/:aid", h.GetApproval)
	approvals.Post("/:aid/approve", h.ApproveApproval)
	approvals.Post("/:aid/reject", h.RejectApproval)
	approvals.Post("/:aid/cancel", h.CancelApproval)
	approvals.Get("/:aid/responses", h.GetApprovalResponses)

	// Project Approvals
	projects.Get("/:id/approvals", h.ListProjectApprovals)

	// Project Schedules
	projects.Get("/:id/schedules", h.ListProjectSchedules)

	// Pipeline Schedules (top-level pipeline routes for schedule management)
	pipelineSchedules := protected.Group("/pipelines")
	pipelineSchedules.Get("/:id/schedules", h.ListPipelineSchedules)
	pipelineSchedules.Post("/:id/schedules", middleware.RequireDev(), h.CreateSchedule)

	// Pipeline Links (composition) and DAG
	pipelineSchedules.Get("/:id/links", h.ListPipelineLinks)
	pipelineSchedules.Post("/:id/links", middleware.RequireDev(), h.CreatePipelineLink)
	pipelineSchedules.Get("/:id/dag", h.GetPipelineDAG)

	// Pipeline Link operations (standalone by link ID)
	pipelineLinksOps := protected.Group("/pipeline-links")
	pipelineLinksOps.Put("/:lid", middleware.RequireDev(), h.UpdatePipelineLink)
	pipelineLinksOps.Delete("/:lid", middleware.RequireAdmin(), h.DeletePipelineLink)

	// Schedule operations (standalone by schedule ID)
	schedules := protected.Group("/schedules")
	schedules.Put("/:sid", middleware.RequireDev(), h.UpdateSchedule)
	schedules.Delete("/:sid", middleware.RequireAdmin(), h.DeleteSchedule)
	schedules.Post("/:sid/enable", middleware.RequireDev(), h.EnableSchedule)
	schedules.Post("/:sid/disable", middleware.RequireDev(), h.DisableSchedule)
	schedules.Get("/:sid/next-runs", h.GetNextRuns)

	// Global search
	protected.Get("/search", h.GlobalSearch)

	// Notification Inbox (in-app notifications)
	inbox := protected.Group("/notifications/inbox")
	inbox.Get("/", h.ListInAppNotifications)
	inbox.Get("/unread-count", h.CountUnreadNotifications)
	inbox.Post("/read-all", h.MarkAllNotificationsRead)
	inbox.Post("/:nid/read", h.MarkNotificationRead)
	inbox.Delete("/:nid", h.DeleteNotification)

	// Notification Preferences
	notifPrefs := protected.Group("/notifications/preferences")
	notifPrefs.Get("/", h.GetNotificationPreferences)
	notifPrefs.Put("/", h.UpdateNotificationPreferences)

	// Dashboard Preferences
	dashPrefs := protected.Group("/dashboard/preferences")
	dashPrefs.Get("/", h.GetDashboardPreferences)
	dashPrefs.Put("/", h.UpdateDashboardPreferences)

	// Auto-Scaling
	scaling := protected.Group("/scaling")
	scaling.Get("/policies", h.ListScalingPolicies)
	scaling.Post("/policies", middleware.RequireAdmin(), h.CreateScalingPolicy)
	scaling.Get("/policies/:pid", h.GetScalingPolicy)
	scaling.Put("/policies/:pid", middleware.RequireAdmin(), h.UpdateScalingPolicy)
	scaling.Delete("/policies/:pid", middleware.RequireAdmin(), h.DeleteScalingPolicy)
	scaling.Post("/policies/:pid/enable", middleware.RequireAdmin(), h.EnableScalingPolicy)
	scaling.Post("/policies/:pid/disable", middleware.RequireAdmin(), h.DisableScalingPolicy)
	scaling.Get("/policies/:pid/events", h.ListScalingEvents)
	scaling.Get("/events", h.ListRecentScalingEvents)
	scaling.Get("/metrics", h.GetScalingMetrics)

	// Global IP Allowlist management (admin only)
	ipAllowlist := protected.Group("/ip-allowlist")
	ipAllowlist.Get("/", middleware.RequireAdmin(), h.ListGlobalIPAllowlist)
	ipAllowlist.Post("/", middleware.RequireAdmin(), h.CreateIPAllowlistEntry)
	ipAllowlist.Delete("/:entryId", middleware.RequireAdmin(), h.DeleteIPAllowlistEntry)

	// Secret scanning
	protected.Post("/security/scan-secrets", middleware.RequireAdmin(), h.ScanRepositoryForSecrets)
	protected.Post("/security/scan-text", middleware.RequireAdmin(), h.ScanTextForSecrets)

	// Overdue secret rotation (global view, admin only)
	protected.Get("/security/rotation/overdue", middleware.RequireAdmin(), h.ListOverdueSecrets)

	// IP allowlist for webhook endpoints
	ipAllow := middleware.NewIPAllowlist(db)
	_ = ipAllow.Reload(context.Background())

	// Webhooks (unauthenticated, signature-validated, IP allowlist enforced)
	webhooks := app.Group("/webhooks", ipAllow.CheckGlobal())
	webhooks.Post("/github", h.GithubWebhook)
	webhooks.Post("/gitlab", h.GitlabWebhook)
	webhooks.Post("/bitbucket", h.BitbucketWebhook)

	// Public badge endpoint (no auth — for embedding in READMEs)
	api.Get("/badges/pipeline/:id", h.GetPipelineBadge)

	// OpenAPI spec
	api.Get("/openapi.json", OpenAPISpec())

	// Templates (public listing, auth for create/update/delete)
	templateRoutes := api.Group("/templates")
	templateRoutes.Get("/", h.ListTemplates)
	templateRoutes.Get("/:id", h.GetTemplate)
	templateRoutes.Post("/", middleware.Auth(cfg.JWTSecret), middleware.RequireDev(), h.CreateTemplate)
	templateRoutes.Put("/:id", middleware.Auth(cfg.JWTSecret), middleware.RequireDev(), h.UpdateTemplate)
	templateRoutes.Delete("/:id", middleware.Auth(cfg.JWTSecret), middleware.RequireAdmin(), h.DeleteTemplate)

	// Security scan results
	protected.Get("/runs/:rid/security", h.GetRunSecurityResults)

	// Admin routes
	adminRoutes := protected.Group("/admin", middleware.RequireAdmin())

	// Feature Flags
	adminRoutes.Get("/features", h.ListFeatureFlags)
	adminRoutes.Post("/features", h.CreateFeatureFlag)
	adminRoutes.Put("/features/:id", h.UpdateFeatureFlag)

	// Dead-Letter Queue
	adminRoutes.Get("/dlq", h.ListDLQ)
	adminRoutes.Post("/dlq/:id/retry", h.RetryDLQ)
	adminRoutes.Delete("/dlq/:id", h.PurgeDLQ)

	// Backup & Restore
	adminRoutes.Post("/backup", h.CreateBackup)
	adminRoutes.Get("/backups", h.ListBackups)
	adminRoutes.Post("/restore", h.RestoreBackup)

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
