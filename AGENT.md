# Agents in FlowForge — Purpose and User Workflow

## What Agents Are

Agents are **remote build workers** that execute CI/CD pipeline jobs on behalf of the FlowForge server. They follow a distributed architecture where the central server orchestrates work and agents perform the actual execution (running build steps, tests, deployments, etc.).

The core model is defined in [`Agent`](backend/internal/models/agent.go:5) with fields for identity, labels, executor type, hardware specs, and status:

| Field          | Description                                    |
|----------------|------------------------------------------------|
| `ID`           | Unique identifier for the agent                |
| `Name`         | Human-readable agent name                      |
| `TokenHash`    | Hashed authentication token (never exposed)    |
| `Labels`       | Comma-separated labels for job routing         |
| `Executor`     | Execution backend: `local`, `docker`, or `kubernetes` |
| `Status`       | Current state: `online`, `offline`, `busy`, `draining` |
| `Version`      | Agent software version                         |
| `OS` / `Arch`  | Operating system and CPU architecture          |
| `CPUCores`     | Number of CPU cores                            |
| `MemoryMB`     | Available memory in megabytes                  |
| `IPAddress`    | Network address of the agent machine           |
| `LastSeenAt`   | Timestamp of last heartbeat                    |
| `CreatedAt`    | When the agent was registered                  |

---

## Architecture Overview

The agent system has four key server-side components:

### 1. Pool ([`pool.go`](backend/internal/agent/pool.go:32))

Manages all connected agents in memory. Tracks status (`online`, `busy`, `draining`, `offline`), resource usage, and selects the least-loaded agent matching a job's requirements via [`SelectAgent()`](backend/internal/agent/pool.go:155).

Key capabilities:
- [`Register()`](backend/internal/agent/pool.go:45) — adds or updates an agent in the pool
- [`Available()`](backend/internal/agent/pool.go:139) — returns agents that can accept new jobs (`online` and below `MaxJobs`)
- [`SelectAgent()`](backend/internal/agent/pool.go:155) — picks the best agent by matching executor type, labels, and lowest load ratio
- [`IncrementActiveJobs()` / `DecrementActiveJobs()`](backend/internal/agent/pool.go:196) — tracks job concurrency; auto-transitions between `online` and `busy` status
- [`MarkOffline()`](backend/internal/agent/pool.go:232) — marks agents as offline if they haven't been seen within the heartbeat timeout

### 2. Dispatcher ([`dispatcher.go`](backend/internal/agent/dispatcher.go:54))

Matches queued [`JobRequest`](backend/internal/agent/dispatcher.go:14) items to available agents. It uses a retry mechanism (up to 3 attempts with 5-second delay) and dispatches jobs based on executor type and label matching.

Key capabilities:
- [`Enqueue()`](backend/internal/agent/dispatcher.go:92) — adds a job request to the dispatch queue (capacity: 256 by default)
- [`Start()`](backend/internal/agent/dispatcher.go:109) — begins the dispatch loop, consuming from the queue
- [`dispatch()`](backend/internal/agent/dispatcher.go:124) — selects a suitable agent, increments its job count, and calls the registered handler

A [`JobRequest`](backend/internal/agent/dispatcher.go:14) includes:
- `JobRunID` and `PipelineRunID` for tracking
- `ExecutorType` and `RequiredLabels` for agent matching
- `Image`, `CloneURL`, `CommitSHA`, `Branch` for repository context
- `EnvVars` and `Steps` (each with command, working directory, timeout, retry count)
- `Priority` for ordering

### 3. Heartbeat Monitor ([`heartbeat.go`](backend/internal/agent/heartbeat.go:11))

Periodically checks agent liveness. Agents that miss heartbeats beyond a 30-second timeout are marked offline. The monitor runs a check every 10 seconds and invokes an optional `onEvict` callback when agents go offline.

### 4. Agent Server ([`server.go`](backend/internal/agent/server.go:17))

HTTP endpoints for agent communication:

| Endpoint                       | Method | Purpose                                    |
|-------------------------------|--------|--------------------------------------------|
| `/api/v1/agents/register`     | POST   | Agent registration with token validation   |
| `/api/v1/agents/heartbeat`    | POST   | Health pings from agents                   |
| `/api/v1/agents/poll`         | POST   | Agents poll for available jobs             |
| `/api/v1/agents/log`          | POST   | Real-time log streaming from agent steps   |
| `/api/v1/agents/complete`     | POST   | Job completion reporting with step results |

---

## How Users Use Agents

### Step 1: Register an Agent via the UI

On the [`AgentsPage`](frontend/src/pages/agents/AgentsPage.tsx:24), users click **"Register Agent"**, provide:
- **Agent Name** — e.g., `agent-linux-04`
- **Executor Type** — `docker`, `kubernetes`, or `local`
- **Labels** — comma-separated tags like `linux, amd64, gpu`

The server creates the agent record in the database ([`008_create_agents.sql`](backend/internal/db/migrations/008_create_agents.sql:2)) and returns a **one-time authentication token** that will not be shown again.

### Step 2: Deploy the Agent Binary on a Worker Machine

Users run the `flowforge-agent` binary ([`cmd/agent/main.go`](backend/cmd/agent/main.go:35)) on any machine where they want builds to execute. Configuration is passed via CLI flags or environment variables:

```bash
flowforge-agent \
  --server https://flowforge.example.com \
  --token <TOKEN> \
  --name agent-linux-04 \
  --executor docker \
  --labels linux,amd64
```

| CLI Flag       | Environment Variable            | Default                  | Description                          |
|---------------|--------------------------------|--------------------------|--------------------------------------|
| `--server`    | `FLOWFORGE_SERVER_URL`          | `http://localhost:8082`  | FlowForge server URL                |
| `--token`     | `FLOWFORGE_AGENT_TOKEN`         | *(required)*             | Agent authentication token           |
| `--name`      | `FLOWFORGE_AGENT_NAME`          | Machine hostname         | Agent display name                   |
| `--executor`  | `FLOWFORGE_AGENT_EXECUTOR`      | `local`                  | Executor type                        |
| `--labels`    | `FLOWFORGE_AGENT_LABELS`        | *(empty)*                | Comma-separated labels               |
| `--log-level` | `FLOWFORGE_LOG_LEVEL`           | `info`                   | Log verbosity                        |

Agents can also be containerized using [`Dockerfile.agent`](deploy/docker/Dockerfile.agent:1), which includes `git` and `docker-cli` in the runtime image and mounts the Docker socket for Docker-in-Docker execution.

### Step 3: Agent Self-Registers with the Server

On startup, the agent calls [`register()`](backend/cmd/agent/main.go:132) which POSTs its capabilities to the server:
- Operating system and architecture (`runtime.GOOS`, `runtime.GOARCH`)
- CPU core count (`runtime.NumCPU()`)
- Executor type and labels
- Agent version (`1.0.0`)

The server validates the token against the `agents` table, registers the agent in the [`Pool`](backend/internal/agent/pool.go:32), sets status to `online`, and returns a heartbeat interval (10 seconds).

### Step 4: Agent Polls for and Executes Jobs

The agent runs two background loops:

**Heartbeat Loop** ([`heartbeatLoop()`](backend/cmd/agent/main.go:188)):
- Sends health pings every 10 seconds
- Reports `active_jobs`, `cpu_usage`, `memory_usage`
- Receives server commands (e.g., `drain` to stop accepting new work)

**Job Loop** ([`jobLoop()`](backend/cmd/agent/main.go:238)):
- Polls for work every 5 seconds via [`pollForJob()`](backend/cmd/agent/main.go:252)
- When a job is received, spawns [`executeJob()`](backend/cmd/agent/main.go:302) in a goroutine

**Job Execution Flow** ([`executeJob()`](backend/cmd/agent/main.go:302)):
1. Iterates through each step in the job
2. Creates an [`ExecutionStep`](backend/internal/engine/executor/executor.go:10) with command, working directory, environment, and timeout
3. If the executor supports [`StreamingExecutor`](backend/internal/engine/executor/executor.go:36), streams logs in real-time to the server via [`sendLog()`](backend/cmd/agent/main.go:390)
4. Tracks step results (status, exit code, duration, errors)
5. Stops on first failure (unless configured otherwise)
6. Reports final status via [`reportJobComplete()`](backend/cmd/agent/main.go:415)

### Step 5: Monitor and Manage via the UI

The [`AgentsPage`](frontend/src/pages/agents/AgentsPage.tsx:24) provides:

- **Status summary cards** — counts of online, busy, draining, and offline agents (clickable to filter)
- **Agent list** — shows each agent with name, status badge, platform (`os/arch`), executor type, version, IP address, CPU/RAM specs, labels, and last seen time
- **Search** — filter agents by name or label
- **Agent detail modal** — full hardware and configuration details for a selected agent
- **Drain action** — gracefully stops new job assignment while letting active jobs finish; the server sends a `drain` command via the heartbeat response
- **Remove action** — deletes the agent from the pool and database

---

## Job Routing

The [`Dispatcher`](backend/internal/agent/dispatcher.go:54) routes jobs to agents using a multi-criteria matching algorithm:

1. **Executor type matching** — a job requiring `docker` execution only goes to agents configured with the `docker` executor
2. **Label matching** — a job requiring `["linux", "gpu"]` only goes to agents that have *both* labels. Checked via [`hasAllLabels()`](backend/internal/agent/pool.go:271)
3. **Load balancing** — among matching candidates, the agent with the lowest `active_jobs / max_jobs` ratio is selected (least-loaded first)
4. **Retry logic** — if no agent matches, the dispatcher retries up to 3 times with 5-second intervals before failing

The `max_jobs` for an agent defaults to its CPU core count (minimum 2), set during [`Register()`](backend/internal/agent/pool.go:45).

---

## Executor Types

Each agent supports one of three execution backends defined in [`NewExecutor()`](backend/internal/engine/executor/executor.go:44):

### Local ([`local.go`](backend/internal/engine/executor/local.go))
- Runs commands directly as OS processes
- Simplest option, no container overhead
- Best for lightweight tasks or development

### Docker ([`docker.go`](backend/internal/engine/executor/docker.go))
- Runs steps inside Docker containers
- Provides isolation between builds
- Requires Docker daemon access (socket mount)
- Supports streaming log output

### Kubernetes ([`kubernetes.go`](backend/internal/engine/executor/kubernetes.go))
- Runs steps as Kubernetes pods
- Best for scalable, cloud-native deployments
- Supports resource limits, node selectors, and pod scheduling

All executors implement the [`Executor`](backend/internal/engine/executor/executor.go:31) interface with an `Execute()` method. Docker and Kubernetes additionally implement [`StreamingExecutor`](backend/internal/engine/executor/executor.go:36) for real-time log output via `ExecuteWithLogs()`.

---

## Agent Lifecycle States

```
┌──────────┐    register    ┌──────────┐    max jobs    ┌──────────┐
│ offline  │ ─────────────> │  online  │ ─────────────> │   busy   │
└──────────┘                └──────────┘                └──────────┘
     ^                           │                           │
     │                           │ drain cmd                 │ jobs finish
     │                           v                           │
     │                      ┌──────────┐                     │
     │                      │ draining │ <───────────────────┘
     │                      └──────────┘
     │                           │
     │     heartbeat timeout     │ all jobs done
     └───────────────────────────┘
```

- **offline** — Agent is not connected or has missed heartbeat timeout (30s)
- **online** — Agent is connected and accepting jobs
- **busy** — Agent is at maximum concurrent job capacity
- **draining** — Agent is finishing active jobs but not accepting new ones (graceful shutdown)

---

## Database Schema

The agents table ([`008_create_agents.sql`](backend/internal/db/migrations/008_create_agents.sql:2)):

```sql
CREATE TABLE agents (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    labels      TEXT NOT NULL DEFAULT '{}',
    executor    TEXT NOT NULL DEFAULT 'local' CHECK(executor IN ('local','docker','kubernetes')),
    status      TEXT NOT NULL DEFAULT 'offline' CHECK(status IN ('online','offline','busy','draining')),
    version     TEXT,
    os          TEXT,
    arch        TEXT,
    cpu_cores   INTEGER,
    memory_mb   INTEGER,
    ip_address  TEXT,
    last_seen_at DATETIME,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## Quick Start Summary

1. **Register** an agent in the FlowForge UI → get a token
2. **Deploy** the `flowforge-agent` binary on a worker machine with the token
3. **Agent connects**, registers capabilities, and starts polling for jobs
4. **Pipeline runs** are automatically routed to matching agents based on executor type and labels
5. **Monitor** agent health, drain gracefully, or remove via the UI
