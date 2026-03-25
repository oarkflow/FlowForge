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
	"github.com/oarkflow/deploy/backend/internal/api"
	"github.com/oarkflow/deploy/backend/internal/api/middleware"
	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/db"
	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/engine"
	"github.com/oarkflow/deploy/backend/internal/importer"
	"github.com/oarkflow/deploy/backend/internal/notifications"
	"github.com/oarkflow/deploy/backend/internal/storage"
	ws "github.com/oarkflow/deploy/backend/internal/websocket"
	"github.com/oarkflow/deploy/backend/internal/worker"
	pkglogger "github.com/oarkflow/deploy/backend/pkg/logger"
)

func main() {
	cfg := config.Load()

	appLog := pkglogger.New(cfg.LogLevel)
	appLog.Info().Str("port", cfg.Port).Msg("starting FlowForge server")

	// Database
	database := db.Connect(cfg.DatabasePath)
	defer database.Close()

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

	// Agent System (pool, dispatcher, heartbeat monitor, server)
	agentPool := agentpkg.NewPool()
	agentDispatcher := agentpkg.NewDispatcher(agentPool, 256)
	agentServer := agentpkg.NewServer(agentPool, agentDispatcher, database)

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
	workerPool := worker.NewPool(database)
	workerPool.SetEngine(eng)
	workerPool.RegisterDefaults()
	workerCtx, workerCancel := context.WithCancel(context.Background())
	go workerPool.Start(workerCtx)

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
	app.Use(middleware.RateLimiter(100, time.Minute))

	// Register REST API routes
	api.RegisterRoutes(app, database, cfg, importSvc, agentServer, eng)

	// WebSocket routes
	app.Get("/ws/runs/:runId/logs", wsHandler.HandleRunLogs)
	app.Get("/ws/events", wsHandler.HandleEvents)

	// SSE fallback
	app.Get("/sse/runs/:runId/logs", wsHandler.SSEHandler)

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
		Str("db", cfg.DatabasePath).
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

	if err := app.Shutdown(); err != nil {
		appLog.Error().Err(err).Msg("shutdown error")
	}

	_ = os.Stdout.Sync()
	appLog.Info().Msg("FlowForge server stopped")
}
