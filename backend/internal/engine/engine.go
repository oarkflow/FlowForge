package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/oarkflow/deploy/backend/internal/config"
	"github.com/oarkflow/deploy/backend/internal/db/queries"
	"github.com/oarkflow/deploy/backend/internal/detector"
	"github.com/oarkflow/deploy/backend/internal/engine/queue"
	"github.com/oarkflow/deploy/backend/internal/engine/scheduler"
	"github.com/oarkflow/deploy/backend/internal/models"
	"github.com/oarkflow/deploy/backend/internal/pipeline"
	"github.com/oarkflow/deploy/backend/internal/websocket"
)

// Engine is the central pipeline execution engine. It manages the job queue,
// scheduler, and runner to orchestrate pipeline execution.
type Engine struct {
	db          *sqlx.DB
	hub         *websocket.Hub
	cfg         *config.Config
	repos       *queries.Repositories
	queue       *queue.PriorityQueue
	scheduler   *scheduler.Scheduler
	runner      *Runner
	composition *pipeline.CompositionService

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Engine with the given database, WebSocket hub, and configuration.
func New(db *sqlx.DB, hub *websocket.Hub, cfg *config.Config) *Engine {
	repos := queries.NewRepositories(db)
	q := queue.NewPriorityQueue()
	sched := scheduler.New(q, db, hub)
	runner := NewRunner(repos, hub)

	e := &Engine{
		db:        db,
		hub:       hub,
		cfg:       cfg,
		repos:     repos,
		queue:     q,
		scheduler: sched,
		runner:    runner,
	}

	// Wire the scheduler to use the runner for executing pipelines
	sched.SetRunFunc(runner.RunPipeline)

	// Set up composition service with a trigger function that calls back into this engine.
	// The trigger function wraps TriggerPipeline to ignore the returned run (we just need error).
	triggerFn := func(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) error {
		_, err := e.TriggerPipeline(ctx, pipelineID, triggerType, triggerData)
		return err
	}
	e.composition = pipeline.NewCompositionService(repos, triggerFn)

	// Wire the scheduler's completion callback for pipeline composition
	sched.SetOnCompleteFunc(e.onRunComplete)

	return e
}

// onRunComplete is called when a pipeline run finishes. It handles
// downstream pipeline triggers via the composition service.
func (e *Engine) onRunComplete(ctx context.Context, runID, status string) {
	// Look up the run to get the pipeline ID
	run, err := e.repos.Runs.GetByID(ctx, runID)
	if err != nil {
		log.Error().Err(err).Str("run_id", runID).Msg("engine: failed to look up run for composition")
		return
	}

	// Extract chain depth from trigger data to prevent infinite loops
	depth := 0
	if run.TriggerData != nil {
		var td map[string]string
		if json.Unmarshal([]byte(*run.TriggerData), &td) == nil {
			if d, ok := td["chain_depth"]; ok {
				fmt.Sscanf(d, "%d", &depth)
			}
		}
	}

	// Trigger downstream pipelines asynchronously
	go func() {
		if err := e.composition.TriggerDownstream(context.Background(), run.PipelineID, status, nil, depth); err != nil {
			log.Error().Err(err).
				Str("pipeline_id", run.PipelineID).
				Str("run_id", runID).
				Msg("engine: downstream trigger failed")
		}
	}()
}

// Start begins the engine's background goroutines (scheduler loop).
// It is non-blocking; call Stop() to shut down.
func (e *Engine) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		log.Info().Msg("engine: starting scheduler")
		e.scheduler.Start(ctx)
	}()

	log.Info().Msg("engine: started")
}

// Stop gracefully shuts down the engine, waiting for in-progress work to complete.
func (e *Engine) Stop() {
	log.Info().Msg("engine: stopping")
	if e.cancel != nil {
		e.cancel()
	}
	e.queue.Close()
	e.wg.Wait()
	log.Info().Msg("engine: stopped")
}

// TriggerPipeline creates a new pipeline run and enqueues it for execution.
func (e *Engine) TriggerPipeline(ctx context.Context, pipelineID, triggerType string, triggerData map[string]string) (*models.PipelineRun, error) {
	// Fetch the pipeline
	pipeline, err := e.repos.Pipelines.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	if pipeline.IsActive == 0 {
		return nil, fmt.Errorf("pipeline %q is not active", pipeline.Name)
	}

	// Get next run number
	number, err := e.repos.Runs.GetNextNumber(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next run number: %w", err)
	}

	// Serialize trigger data
	var triggerDataJSON *string
	if len(triggerData) > 0 {
		data, err := json.Marshal(triggerData)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize trigger data: %w", err)
		}
		s := string(data)
		triggerDataJSON = &s
	}

	// Create the pipeline run record
	run := &models.PipelineRun{
		PipelineID:  pipelineID,
		Number:      number,
		Status:      "queued",
		TriggerType: triggerType,
		TriggerData: triggerDataJSON,
	}

	// Extract common trigger data fields
	if branch, ok := triggerData["branch"]; ok {
		run.Branch = &branch
	}
	if sha, ok := triggerData["commit_sha"]; ok {
		run.CommitSHA = &sha
	}
	if msg, ok := triggerData["commit_message"]; ok {
		run.CommitMessage = &msg
	}
	if author, ok := triggerData["author"]; ok {
		run.Author = &author
	}
	if tag, ok := triggerData["tag"]; ok {
		run.Tag = &tag
	}

	if err := e.repos.Runs.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create pipeline run: %w", err)
	}

	// Build the pipeline config for the queue job.
	// If the pipeline has config_content, use it; otherwise use a default.
	pipelineConfig := e.buildPipelineConfig(ctx, pipeline, triggerData)

	configJSON, err := json.Marshal(pipelineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize pipeline config: %w", err)
	}

	// Enqueue the job
	job := &queue.Job{
		ID:            uuid.New().String(),
		PipelineRunID: run.ID,
		Priority:      determinePriority(triggerType),
		CreatedAt:     time.Now(),
		Config:        configJSON,
	}

	if err := e.queue.Enqueue(job); err != nil {
		// If enqueue fails, mark the run as failed
		_ = e.repos.Runs.UpdateStatus(ctx, run.ID, "failure")
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	log.Info().
		Str("pipeline_id", pipelineID).
		Str("run_id", run.ID).
		Int("number", run.Number).
		Str("trigger", triggerType).
		Msg("engine: pipeline triggered")

	return run, nil
}

// buildPipelineConfig converts the pipeline's stored config into a scheduler.PipelineConfig.
// It supports both the FlowForge YAML DSL and direct JSON scheduler config.
// It also injects repository context (clone URL, branch) into step env vars.
//
// For pipelines linked to a local repository, it re-runs detection on every
// trigger so that changes in autogen.go (e.g. updated Docker images) take
// effect immediately without requiring the user to manually update the config.
func (e *Engine) buildPipelineConfig(ctx context.Context, p *models.Pipeline, triggerData map[string]string) scheduler.PipelineConfig {
	content := ""
	if p.ConfigContent != nil {
		content = *p.ConfigContent
	}

	// If the pipeline has a linked repository with a local path, re-detect
	// the stack and regenerate the config. This ensures updated Docker images
	// and build commands take effect without manual config edits.
	if p.RepositoryID != nil && *p.RepositoryID != "" {
		repo, err := e.repos.Repos.GetByID(ctx, *p.RepositoryID)
		if err == nil && isLocalPath(repo.CloneURL) {
			results, detectErr := detector.Detect(repo.CloneURL)
			if detectErr == nil && len(results) > 0 {
				newYAML := detector.GenerateStarterPipeline(results)
				log.Info().Str("pipeline_id", p.ID).Msg("engine: regenerated pipeline config from local repo detection")
				content = newYAML
				// Persist the updated config so future views in the UI reflect the change
				p.ConfigContent = &newYAML
				now := time.Now()
				p.UpdatedAt = now
				_ = e.repos.Pipelines.Update(ctx, p)
			}
		}
	}

	if content == "" {
		return defaultPipelineConfig()
	}

	// 1. Try direct JSON unmarshal into scheduler.PipelineConfig (for configs stored as scheduler JSON)
	var direct scheduler.PipelineConfig
	if err := json.Unmarshal([]byte(content), &direct); err == nil && len(direct.Stages) > 0 {
		e.injectProjectEnv(ctx, p, &direct)
		e.injectRepoEnv(ctx, p, triggerData, &direct)
		return direct
	}

	// 2. Parse as FlowForge YAML/JSON DSL using the pipeline parser
	spec, err := pipeline.Parse(content)
	if err != nil {
		log.Warn().Err(err).Str("pipeline_id", p.ID).Msg("engine: failed to parse pipeline config, using default")
		return defaultPipelineConfig()
	}

	config := specToSchedulerConfig(spec)
	e.injectProjectEnv(ctx, p, &config)
	e.injectRepoEnv(ctx, p, triggerData, &config)
	return config
}

// injectProjectEnv fetches project-level environment variables and secrets,
// then injects them into every step's env map. These are set before YAML-level
// env vars, so pipeline/job/step env definitions can override them.
func (e *Engine) injectProjectEnv(ctx context.Context, p *models.Pipeline, config *scheduler.PipelineConfig) {
	// 1. Fetch project env vars (plaintext key-value pairs)
	envVars, err := e.repos.EnvVars.GetForInjection(ctx, p.ProjectID)
	if err != nil {
		log.Warn().Err(err).Str("project_id", p.ProjectID).Msg("engine: failed to fetch project env vars")
	}

	// 2. Fetch project secrets (encrypted, decrypted at runtime)
	secretVars, err := e.repos.Secrets.ListByProject(ctx, p.ProjectID, 10000, 0)
	if err != nil {
		log.Warn().Err(err).Str("project_id", p.ProjectID).Msg("engine: failed to fetch project secrets")
	}

	if len(envVars) == 0 && len(secretVars) == 0 {
		return
	}

	// Inject into every step — env vars first, then any secret keys that
	// aren't already overridden by explicit env vars or pipeline config.
	for i := range config.Stages {
		for j := range config.Stages[i].Jobs {
			for k := range config.Stages[i].Jobs[j].Steps {
				if config.Stages[i].Jobs[j].Steps[k].Env == nil {
					config.Stages[i].Jobs[j].Steps[k].Env = make(map[string]string)
				}
				env := config.Stages[i].Jobs[j].Steps[k].Env

				// Project env vars (lowest priority — can be overridden by pipeline YAML env)
				for key, val := range envVars {
					if _, exists := env[key]; !exists {
						env[key] = val
					}
				}

				// Secrets are decrypted and injected by name. They are also
				// lowest priority so pipeline YAML can override if needed.
				// Note: The encrypted value from DB is not useful here. We
				// need the SecretStore for decryption. For now, we only inject
				// plaintext env vars. Secrets injection requires the
				// SecretStore which lives in the handler layer. We'll add
				// secret injection when the SecretStore is wired into the engine.
			}
		}
	}
}

// injectRepoEnv looks up the repository associated with a pipeline and injects
// FLOWFORGE_REPO_CLONE_URL and FLOWFORGE_REPO_BRANCH into every step's env,
// so that the checkout action knows where to clone from.
func (e *Engine) injectRepoEnv(ctx context.Context, p *models.Pipeline, triggerData map[string]string, config *scheduler.PipelineConfig) {
	var cloneURL, branch string

	// Try to get repo info from the linked repository
	if p.RepositoryID != nil && *p.RepositoryID != "" {
		repo, err := e.repos.Repos.GetByID(ctx, *p.RepositoryID)
		if err == nil {
			cloneURL = repo.CloneURL
			branch = repo.DefaultBranch
		}
	}

	// Override branch from trigger data if available
	if b, ok := triggerData["branch"]; ok && b != "" {
		branch = b
	}
	if branch == "" {
		branch = "main"
	}

	if cloneURL == "" {
		return // No repo linked, checkout will skip gracefully
	}

	// Inject into every step
	for i := range config.Stages {
		for j := range config.Stages[i].Jobs {
			for k := range config.Stages[i].Jobs[j].Steps {
				if config.Stages[i].Jobs[j].Steps[k].Env == nil {
					config.Stages[i].Jobs[j].Steps[k].Env = make(map[string]string)
				}
				config.Stages[i].Jobs[j].Steps[k].Env["FLOWFORGE_REPO_CLONE_URL"] = cloneURL
				config.Stages[i].Jobs[j].Steps[k].Env["FLOWFORGE_REPO_BRANCH"] = branch
			}
		}
	}
}

// resolveAction converts a `uses: flowforge/<action>@<version>` into an executable
// shell command. The action's `with` parameters are injected as env vars prefixed
// with INPUT_.
func resolveAction(uses string, with map[string]string) string {
	// Parse "flowforge/<action>@<version>"
	action := uses
	if idx := strings.Index(action, "@"); idx != -1 {
		action = action[:idx]
	}
	action = strings.TrimPrefix(action, "flowforge/")

	switch action {
	case "checkout":
		// Clone the repository or copy a local folder. Relies on
		// FLOWFORGE_REPO_CLONE_URL and FLOWFORGE_REPO_BRANCH being set in the
		// step env by the config builder. Handles three cases:
		//  1. Local folder WITHOUT .git → copy contents (cp -a)
		//  2. Local folder WITH .git → git clone (file:// protocol)
		//  3. Remote URL (https:// or git@) → git clone
		return `set -e
REPO_URL="${INPUT_REPOSITORY:-${FLOWFORGE_REPO_CLONE_URL:-}}"
BRANCH="${INPUT_BRANCH:-${FLOWFORGE_REPO_BRANCH:-main}}"
DEPTH="${INPUT_DEPTH:-1}"

# Skip checkout if workspace already has files (shared volume from prior stage)
FILE_COUNT=$(ls -1A /workspace 2>/dev/null | wc -l)
if [ "$FILE_COUNT" -gt 0 ]; then
  echo "Workspace already has $FILE_COUNT entries — skipping checkout"
  exit 0
fi

if [ -z "$REPO_URL" ]; then
  echo "Checkout: No repository URL configured — skipping"
  exit 0
  exit 0
fi

# Helper: copy all files (including hidden) from $1 into current directory
copy_source() {
  SRC="$1"
  # Use find + cp to reliably copy all files including hidden ones
  if command -v rsync >/dev/null 2>&1; then
    rsync -a "$SRC/" ./
  else
    # tar pipe is the most reliable cross-platform approach
    (cd "$SRC" && tar cf - .) | tar xf -
  fi
}

# Determine if this is a local path or remote URL
IS_LOCAL=false
LOCAL_PATH=""
case "$REPO_URL" in
  /*) IS_LOCAL=true; LOCAL_PATH="$REPO_URL" ;;
  ./*|../*) IS_LOCAL=true; LOCAL_PATH="$REPO_URL" ;;
esac

if [ "$IS_LOCAL" = "true" ]; then
  if [ ! -e "$LOCAL_PATH" ]; then
    # Inside a container, local paths are mounted at /mnt/source
    if [ -e "/mnt/source" ]; then
      copy_source /mnt/source
      exit 0
    fi
    echo "Warning: Local path '$LOCAL_PATH' not found — skipping checkout"
    exit 0
  fi
  if [ -d "$LOCAL_PATH/.git" ]; then
    # Local git repo — clone it
    echo "Cloning local git repository: $LOCAL_PATH (branch: $BRANCH)"
    if ! command -v git >/dev/null 2>&1; then
      if command -v apk >/dev/null 2>&1; then apk add --no-cache git >/dev/null 2>&1
      elif command -v apt-get >/dev/null 2>&1; then apt-get update -qq && apt-get install -y -qq git >/dev/null 2>&1
      elif command -v yum >/dev/null 2>&1; then yum install -y -q git >/dev/null 2>&1; fi
    fi
    if command -v git >/dev/null 2>&1; then
      git clone --depth="$DEPTH" --branch "$BRANCH" --single-branch "file://$LOCAL_PATH" . 2>&1 || \
        git clone "file://$LOCAL_PATH" . 2>&1 || \
        { echo "git clone failed, falling back to copy"; copy_source "$LOCAL_PATH"; }
    else
      echo "git not available, copying files from $LOCAL_PATH"
      copy_source "$LOCAL_PATH"
    fi
  else
    # Local directory without git — just copy the files
    echo "Copying local source from: $LOCAL_PATH"
    copy_source "$LOCAL_PATH"
  fi
  echo "Source files ready in workspace"
else
  # Remote URL — use git clone
  echo "Cloning $REPO_URL (branch: $BRANCH, depth: $DEPTH)"
  if ! command -v git >/dev/null 2>&1; then
    if command -v apk >/dev/null 2>&1; then apk add --no-cache git >/dev/null 2>&1
    elif command -v apt-get >/dev/null 2>&1; then apt-get update -qq && apt-get install -y -qq git >/dev/null 2>&1
    elif command -v yum >/dev/null 2>&1; then yum install -y -q git >/dev/null 2>&1; fi
  fi
  if [ -d ".git" ]; then
    echo "Repository already present, fetching latest..."
    git fetch --depth="$DEPTH" origin "$BRANCH"
    git checkout FETCH_HEAD
  else
    git clone --depth="$DEPTH" --branch "$BRANCH" --single-branch "$REPO_URL" .
  fi
  echo "Checked out $(git rev-parse --short HEAD 2>/dev/null || echo 'N/A') on $BRANCH"
fi`

	case "upload-artifact":
		name := with["name"]
		path := with["path"]
		if name == "" {
			name = "artifact"
		}
		if path == "" {
			path = "."
		}
		// For now, create a tarball in /tmp/flowforge-artifacts/ so it's preserved.
		return fmt.Sprintf(`set -e
ARTIFACT_NAME="%s"
ARTIFACT_PATH="%s"
ARTIFACT_DIR="/tmp/flowforge-artifacts"
mkdir -p "$ARTIFACT_DIR"
if [ -e "$ARTIFACT_PATH" ]; then
  tar czf "$ARTIFACT_DIR/${ARTIFACT_NAME}.tar.gz" -C "$(dirname "$ARTIFACT_PATH")" "$(basename "$ARTIFACT_PATH")"
  echo "Uploaded artifact '$ARTIFACT_NAME' ($(du -sh "$ARTIFACT_DIR/${ARTIFACT_NAME}.tar.gz" | cut -f1))"
else
  echo "Warning: artifact path '$ARTIFACT_PATH' not found — skipping upload"
fi`, name, path)

	case "download-artifact":
		name := with["name"]
		path := with["path"]
		if name == "" {
			name = "artifact"
		}
		if path == "" {
			path = "."
		}
		return fmt.Sprintf(`set -e
ARTIFACT_NAME="%s"
DEST_PATH="%s"
ARTIFACT_DIR="/tmp/flowforge-artifacts"
ARCHIVE="$ARTIFACT_DIR/${ARTIFACT_NAME}.tar.gz"
if [ -f "$ARCHIVE" ]; then
  mkdir -p "$DEST_PATH"
  tar xzf "$ARCHIVE" -C "$DEST_PATH"
  echo "Downloaded artifact '$ARTIFACT_NAME' to $DEST_PATH"
else
  echo "Warning: artifact '$ARTIFACT_NAME' not found — skipping download"
fi`, name, path)

	case "docker-build-push":
		registry := with["registry"]
		image := with["image"]
		tags := with["tags"]
		if tags == "" {
			tags = "latest"
		}
		pushFlag := ""
		if with["push"] == "true" || with["push"] == "" {
			pushFlag = "--push"
		}
		tagArgs := ""
		for _, tag := range strings.Split(tags, "\n") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				fullTag := tag
				if registry != "" && image != "" {
					fullTag = fmt.Sprintf("%s/%s:%s", registry, image, tag)
				}
				tagArgs += fmt.Sprintf(" -t %s", fullTag)
			}
		}
		return fmt.Sprintf(`set -e
echo "Building Docker image..."
docker build%s %s .
echo "Docker build complete"`, tagArgs, pushFlag)

	case "helm-deploy":
		chart := with["chart"]
		release := with["release"]
		namespace := with["namespace"]
		values := with["values"]
		if chart == "" {
			chart = "."
		}
		if release == "" {
			release = "app"
		}
		nsFlag := ""
		if namespace != "" {
			nsFlag = fmt.Sprintf(" --namespace %s", namespace)
		}
		valuesFlag := ""
		if values != "" {
			valuesFlag = " --set " + strings.ReplaceAll(strings.TrimSpace(values), "\n", " --set ")
		}
		return fmt.Sprintf(`set -e
echo "Deploying Helm chart: %s as release %s"
helm upgrade --install %s %s%s%s --wait
echo "Helm deploy complete"`, chart, release, release, chart, nsFlag, valuesFlag)

	default:
		// Unknown action — log it clearly but don't fail
		return fmt.Sprintf(`echo "Warning: unknown action '%s' — no built-in handler found"`, uses)
	}
}

// specToSchedulerConfig converts a parsed PipelineSpec into a scheduler.PipelineConfig
// by grouping jobs by stage and mapping step fields.
func specToSchedulerConfig(spec *pipeline.PipelineSpec) scheduler.PipelineConfig {
	if spec == nil || len(spec.Jobs) == 0 {
		return defaultPipelineConfig()
	}

	// Resolve defaults for executor and image
	defaultExecutor := "docker"
	defaultImage := ""
	if spec.Defaults != nil {
		if spec.Defaults.Executor != "" {
			defaultExecutor = spec.Defaults.Executor
		}
		if spec.Defaults.Image != "" {
			defaultImage = spec.Defaults.Image
		}
	}

	// Determine stage ordering: use explicit stages list if provided,
	// otherwise collect stages from jobs in insertion order (map iteration order is random,
	// so fall back to alphabetical).
	stageOrder := spec.Stages
	if len(stageOrder) == 0 {
		seen := make(map[string]bool)
		for _, job := range spec.Jobs {
			stageName := job.Stage
			if stageName == "" {
				stageName = "default"
			}
			if !seen[stageName] {
				stageOrder = append(stageOrder, stageName)
				seen[stageName] = true
			}
		}
	}

	// Group jobs by stage name
	stageJobs := make(map[string][]scheduler.JobConfig)
	for jobName, jobSpec := range spec.Jobs {
		stageName := jobSpec.Stage
		if stageName == "" {
			stageName = "default"
		}

		// Determine executor type for this job
		executorType := jobSpec.Executor
		if executorType == "" {
			executorType = defaultExecutor
		}

		// Determine Docker image for this job
		jobImage := jobSpec.Image
		if jobImage == "" {
			jobImage = defaultImage
		}

		// Convert steps
		var steps []scheduler.StepConfig
		for i, stepSpec := range jobSpec.Steps {
			// Resolve command from either Run or Uses
			cmd := stepSpec.Run
			if cmd == "" && stepSpec.Uses != "" {
				cmd = resolveAction(stepSpec.Uses, stepSpec.With)
			}
			if cmd == "" {
				continue // Skip steps with no command
			}

			name := stepSpec.Name
			if name == "" {
				name = fmt.Sprintf("step-%d", i+1)
			}

			// Merge env vars: global → job → step → internal
			env := make(map[string]string)
			for k, v := range spec.Env {
				env[k] = v
			}
			for k, v := range jobSpec.Env {
				env[k] = v
			}
			for k, v := range stepSpec.Env {
				env[k] = v
			}

			// Inject action `with` parameters as INPUT_* env vars
			if stepSpec.Uses != "" {
				for k, v := range stepSpec.With {
					env["INPUT_"+strings.ToUpper(k)] = v
				}
			}

			// Pass Docker image to the Docker executor via env
			if executorType == "docker" && jobImage != "" {
				env["FLOWFORGE_DOCKER_IMAGE"] = jobImage
			}

			// Propagate privileged mode: mount the host Docker socket so
			// the container can run docker/docker-compose commands against
			// the host daemon (no DinD required).
			if executorType == "docker" && jobSpec.Privileged {
				env["FLOWFORGE_DOCKER_MOUNT_DOCKER_SOCKET"] = "true"
				env["FLOWFORGE_DOCKER_PRIVILEGED"] = "true"
			}

			steps = append(steps, scheduler.StepConfig{
				Name:    name,
				Command: cmd,
				Env:     env,
				Timeout: jobSpec.Timeout,
			})
		}

		if len(steps) == 0 {
			continue // Skip jobs with no executable steps
		}

		jc := scheduler.JobConfig{
			Name:         jobName,
			ExecutorType: executorType,
			Steps:        steps,
		}

		stageJobs[stageName] = append(stageJobs[stageName], jc)
	}

	// Build ordered stage configs, propagating stage-level needs
	var stages []scheduler.StageConfig
	for _, stageName := range stageOrder {
		jobs, ok := stageJobs[stageName]
		if !ok || len(jobs) == 0 {
			continue
		}
		sc := scheduler.StageConfig{
			Name: stageName,
			Jobs: jobs,
		}
		if spec.StageNeeds != nil {
			if needs, ok := spec.StageNeeds[stageName]; ok {
				sc.Needs = needs
			}
		}
		stages = append(stages, sc)
	}

	if len(stages) == 0 {
		return defaultPipelineConfig()
	}

	return scheduler.PipelineConfig{Stages: stages}
}

// defaultPipelineConfig returns a minimal pipeline config that just echoes a message.
func defaultPipelineConfig() scheduler.PipelineConfig {
	return scheduler.PipelineConfig{
		Stages: []scheduler.StageConfig{
			{
				Name: "default",
				Jobs: []scheduler.JobConfig{
					{
						Name:         "build",
						ExecutorType: "docker",
						Steps: []scheduler.StepConfig{
							{
								Name:    "echo",
								Command: "echo 'No pipeline configuration found'",
							},
						},
					},
				},
			},
		},
	}
}

// determinePriority assigns a priority based on trigger type.
// Manual triggers get highest priority, followed by API and push.
func determinePriority(triggerType string) int {
	switch triggerType {
	case "manual":
		return 100
	case "api":
		return 80
	case "push":
		return 50
	case "pull_request":
		return 50
	case "schedule":
		return 30
	case "pipeline":
		return 60
	default:
		return 50
	}
}

// isLocalPath returns true if the URL looks like a local filesystem path.
func isLocalPath(url string) bool {
	return strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../")
}

// Queue returns the engine's priority queue (for external inspection or testing).
func (e *Engine) Queue() *queue.PriorityQueue {
	return e.queue
}

// ReenqueueRun re-enqueues a pipeline run that is stuck in "queued" status in the DB.
// This is used by the stale run recovery worker to pick up orphaned runs after a restart.
func (e *Engine) ReenqueueRun(ctx context.Context, runID string) error {
	run, err := e.repos.Runs.GetByID(ctx, runID)
	if err != nil {
		return fmt.Errorf("run not found: %w", err)
	}

	if run.Status != "queued" {
		return nil // Already running or finished
	}

	pipeline, err := e.repos.Pipelines.GetByID(ctx, run.PipelineID)
	if err != nil {
		return fmt.Errorf("pipeline not found: %w", err)
	}

	// Reconstruct trigger data from the stored run for repo context injection
	rerunTriggerData := make(map[string]string)
	if run.Branch != nil {
		rerunTriggerData["branch"] = *run.Branch
	}

	pipelineConfig := e.buildPipelineConfig(ctx, pipeline, rerunTriggerData)
	configJSON, err := json.Marshal(pipelineConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	job := &queue.Job{
		ID:            uuid.New().String(),
		PipelineRunID: run.ID,
		Priority:      determinePriority(run.TriggerType),
		CreatedAt:     time.Now(),
		Config:        configJSON,
	}

	if err := e.queue.Enqueue(job); err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	log.Info().
		Str("run_id", runID).
		Msg("engine: re-enqueued stale run")

	return nil
}
