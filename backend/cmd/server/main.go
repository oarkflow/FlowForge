package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	agentpkg "github.com/oarkflow/deploy/backend/internal/agent"
	"github.com/oarkflow/deploy/backend/internal/admin"
	"github.com/oarkflow/deploy/backend/internal/api"
	"github.com/oarkflow/deploy/backend/internal/api/handlers"
	"github.com/oarkflow/deploy/backend/internal/api/middleware"
	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/db"
	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine"
	"github.com/oarkflow/deploy/backend/internal/engine/queue"
	"github.com/oarkflow/deploy/backend/internal/features"
	"github.com/oarkflow/deploy/backend/internal/importer"
	"github.com/oarkflow/deploy/backend/internal/logging"
	"github.com/oarkflow/deploy/backend/internal/notifications"
	"github.com/oarkflow/deploy/backend/internal/scheduler"
	securitypkg "github.com/oarkflow/deploy/backend/internal/security"
	"github.com/oarkflow/deploy/backend/internal/storage"
	"github.com/oarkflow/deploy/backend/internal/templates"
	ws "github.com/oarkflow/deploy/backend/internal/websocket"
	"github.com/oarkflow/deploy/backend/internal/worker"
	pkglogger "github.com/oarkflow/deploy/backend/pkg/logger"
)

func main() {
	cfg := config.Load()

	appLog := pkglogger.New(cfg.LogLevel)
	appLog.Info().Str("port", cfg.Port).Msg("starting FlowForge server")

	// Database
	database, err := db.Connect(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	appLog.Info().Msg("database connection established")

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	appLog.Info().Msg("database migrations completed")

	// Seed default admin user if no users exist
	if err := db.SeedDefaultAdmin(database); err != nil {
		appLog.Warn().Err(err).Msg("failed to seed default admin user")
	}

	// Repositories
	repos := queries.NewRepositories(database)

	// Encryption key
	encKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil || len(encKey) != 32 {
		appLog.Warn().Msg("invalid encryption key, using default (INSECURE — change in production)")
		encKey = make([]byte, 32) // zeroed key for dev
	}

	// WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()
	wsHandler := ws.NewHandler(hub, repos)

	// Pipeline Execution Engine
	eng := engine.New(database, hub, cfg)
	eng.Start()

	// Notification Dispatcher
	notifBus := notifications.NewEventBus(256)
	notifDispatcher := notifications.NewDispatcher(repos, encKey, notifBus)
	notifCtx, notifCancel := context.WithCancel(context.Background())
	go notifDispatcher.Start(notifCtx)

	// Artifact Storage (local disk by default)
	artifactDir := cfg.DatabasePath + ".artifacts"
	localStore := storage.NewLocalBackend(artifactDir)
	_ = storage.NewArtifactService(localStore, repos)

	// Import Service
	importSvc := importer.New(encKey)

	// Agent System (pool, dispatcher, heartbeat monitor, HTTP server, gRPC server)
	agentPool := agentpkg.NewPool()
	agentDispatcher := agentpkg.NewDispatcher(agentPool, 256)
	agentServer := agentpkg.NewServer(agentPool, agentDispatcher, database)

	// gRPC agent server — runs alongside HTTP for agents that prefer gRPC transport
	grpcAgentServer := agentpkg.NewGRPCAgentServer(agentPool, agentDispatcher, database, hub)
	var grpcSrv interface{ Stop() }
	if cfg.GRPCPort != "" {
		srv, lis, err := grpcAgentServer.StartServer(":" + cfg.GRPCPort)
		if err != nil {
			appLog.Error().Err(err).Msg("failed to start gRPC agent server")
		} else {
			grpcSrv = srv
			appLog.Info().Str("grpc_port", cfg.GRPCPort).Str("addr", lis.Addr().String()).Msg("gRPC agent server started")
		}
	}

	heartbeatMonitor := agentpkg.NewHeartbeatMonitor(agentPool, 60*time.Second, 10*time.Second)
	heartbeatMonitor.OnEvict(func(agentID string) {
		appLog.Warn().Str("agent_id", agentID).Msg("agent evicted due to heartbeat timeout")
		// Update DB status
		database.Exec("UPDATE agents SET status = 'offline' WHERE id = ?", agentID)
	})
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	go heartbeatMonitor.Start(heartbeatCtx)

	dispatchCtx, dispatchCancel := context.WithCancel(context.Background())
	go agentDispatcher.Start(dispatchCtx)

	// Background Worker Pool
	schedulerSvc := scheduler.NewService(repos)
	schedulerSvc.SetEngine(eng)

	// Auto-Scaler
	autoScaler := agentpkg.NewAutoScaler(repos, agentPool, agentDispatcher)

	workerPool := worker.NewPool(database)
	workerPool.SetEngine(eng)
	workerPool.SetScheduler(schedulerSvc)
	workerPool.SetAutoScaler(autoScaler)
	workerPool.RegisterDefaults()
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go workerPool.Start(workerCtx)

	// ---- New Services ----

	// Pipeline Template Store
	tmplStore := templates.NewStore(repos.Templates)
	tmplCtx := context.Background()
	if err := tmplStore.SeedBuiltins(tmplCtx); err != nil {
		appLog.Warn().Err(err).Msg("failed to seed builtin templates")
	}
	handlers.SetTemplateStore(tmplStore)

	// Feature Flags
	flagSvc := features.NewFlagService(repos.FeatureFlags)
	handlers.SetFlagService(flagSvc)

	// Dead-Letter Queue
	dlq := queue.NewDeadLetterQueue(repos.DeadLetters)
	handlers.SetDeadLetterQueue(dlq)

	// Backup Service
	backupSvc := admin.NewBackupService(database, cfg.BackupDir)
	handlers.SetBackupService(backupSvc)

	// Security Scan Service
	scanSvc := securitypkg.NewScanService(repos.ScanResults)
	handlers.SetScanService(scanSvc)

	// Log Retention Worker
	retentionCtx, retentionCancel := context.WithCancel(context.Background())
	retentionWorker := logging.NewRetentionWorker(repos, cfg.GlobalLogRetentionDays)
	go retentionWorker.Start(retentionCtx)

	// Log Forwarding Manager
	var logFwdManager *logging.ForwarderManager
	if cfg.LogForwardingEnabled && cfg.LogForwardingType != "" {
		fwdCfg := logging.ForwarderConfig{
			Type:      cfg.LogForwardingType,
			Endpoint:  cfg.LogForwardingURL,
			AuthToken: cfg.LogForwardingToken,
			Index:     cfg.LogForwardingIndex,
		}
		logFwdManager = logging.NewForwarderManager([]logging.ForwarderConfig{fwdCfg})
		fwdCtx, _ := context.WithCancel(context.Background())
		logFwdManager.Start(fwdCtx, 5*time.Second)
		appLog.Info().Str("type", cfg.LogForwardingType).Msg("log forwarding enabled")
	}
	_ = logFwdManager // available for engine integration

	// Scheduled Backup
	if cfg.BackupInterval != "" {
		interval, err := time.ParseDuration(cfg.BackupInterval)
		if err == nil && interval > 0 {
			backupCtx, _ := context.WithCancel(context.Background())
			go backupSvc.ScheduledBackup(backupCtx, interval)
			appLog.Info().Str("interval", cfg.BackupInterval).Msg("scheduled backups enabled")
		}
	}

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "FlowForge v1.0",
		ErrorHandler: api.ErrorHandler,
		BodyLimit:    cfg.MaxUploadSize,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{AllowOrigins: []string{cfg.AllowedOrigins}}))
	app.Use(compress.New())
	app.Use(middleware.RequestLogger(appLog))

	// WebSocket routes — registered BEFORE the rate limiter so real-time
	// connections are not throttled or rejected by per-IP limits.
	app.Get("/ws/runs/:runId/logs", wsHandler.HandleRunLogs)
	app.Get("/ws/events", wsHandler.HandleEvents)
	app.Get("/sse/runs/:runId/logs", wsHandler.SSEHandler)

	// Rate limiter applies only to REST API routes (registered after this point)
	app.Use(middleware.RateLimiter(100, time.Minute))

	// Per-user rate limiter (applied after auth extracts user info)
	userLimiter := middleware.NewUserRateLimiter(middleware.RoleLimits{
		Owner:     2000,
		Admin:     cfg.RateLimitAdmin,
		Developer: cfg.RateLimitDeveloper,
		Viewer:    cfg.RateLimitViewer,
	}, time.Minute)
	app.Use(userLimiter.Handler())

	// Register REST API routes
	api.RegisterRoutes(app, database, cfg, importSvc, agentServer, eng)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			appLog.Fatal().Err(err).Msg("server error")
		}
	}()

	appLog.Info().
		Str("port", cfg.Port).
		Str("grpc_port", cfg.GRPCPort).
		Str("database", cfg.DatabasePath).
		Bool("embedded_worker", cfg.EmbeddedWorker).
		Msg("FlowForge server is running")

	<-ctx.Done()
	appLog.Info().Msg("shutting down...")

	// Graceful shutdown sequence
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = shutdownCtx

	// Stop services
	eng.Stop()
	notifCancel()
	heartbeatCancel()
	dispatchCancel()
	workerCancel()
	retentionCancel()
	if grpcSrv != nil {
		grpcSrv.Stop()
		appLog.Info().Msg("gRPC agent server stopped")
	}
	if logFwdManager != nil {
		logFwdManager.Close()
	}

	if err := app.Shutdown(); err != nil {
		appLog.Error().Err(err).Msg("shutdown error")
	}

	_ = os.Stdout.Sync()
	appLog.Info().Msg("FlowForge server stopped")
}
