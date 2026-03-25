# CLAUDE.md — FlowForge CI/CD Platform

> **Project:** FlowForge — A robust, distributed, cloud-native CI/CD platform
> **Backend:** Go 1.22+ · GoFiber v3.1.0 · jmoiron/sqlx · SQLite (WAL mode)
> **Frontend:** SolidJS · TailwindCSS v4 · TypeScript
> **Deployment:** Docker · Kubernetes · AWS / GCP / Azure / On-Premises

---

## TABLE OF CONTENTS

1. [Project Overview](#1-project-overview)
2. [Complete Feature List](#2-complete-feature-list)
3. [Repository Structure](#3-repository-structure)
4. [Tech Stack & Dependencies](#4-tech-stack--dependencies)
5. [Architecture Overview](#5-architecture-overview)
6. [Subagent Task Breakdown (Parallel Workstreams)](#6-subagent-task-breakdown-parallel-workstreams)
7. [Backend Implementation Guide](#7-backend-implementation-guide)
8. [Frontend Implementation Guide](#8-frontend-implementation-guide)
9. [Database Schema](#9-database-schema)
10. [Pipeline Configuration DSL](#10-pipeline-configuration-dsl)
11. [Security Implementation](#11-security-implementation)
12. [Distributed Execution Engine](#12-distributed-execution-engine)
13. [API Reference Structure](#13-api-reference-structure)
14. [Deployment & Infrastructure](#14-deployment--infrastructure)
15. [Testing Strategy](#15-testing-strategy)
16. [Prompts for Subagents](#16-prompts-for-subagents)

---

## 1. PROJECT OVERVIEW

FlowForge is a self-hosted, horizontally-scalable CI/CD platform that:
- Connects to GitHub, GitLab, and Bitbucket via webhooks and OAuth
- Detects languages/frameworks/dependencies automatically
- Executes pipelines locally, in Docker containers, or on Kubernetes
- Stores all state in SQLite with WAL mode (upgradeable to PostgreSQL via sqlx)
- Exposes a comprehensive REST + WebSocket API
- Provides a real-time SolidJS dashboard

---

## 2. COMPLETE FEATURE LIST

### 2.1 Repository Integration
- [ ] GitHub App / OAuth integration (webhooks: push, PR, release, tag)
- [ ] GitLab integration (webhooks + personal access tokens)
- [ ] Bitbucket integration (webhooks + app passwords)
- [ ] Repository auto-discovery and import
- [ ] Branch filtering rules (glob patterns)
- [ ] Monorepo support with path-based triggers
- [ ] Git submodule support
- [ ] Repository mirroring and caching (shallow clones)
- [ ] Protected branch enforcement in pipelines
- [ ] SSH key management per repository

### 2.2 Pipeline Triggers
- [ ] Push trigger (branch/tag/path filters)
- [ ] Pull Request / Merge Request trigger (open, sync, close, label)
- [ ] Schedule trigger (cron expressions with timezone support)
- [ ] Manual trigger with parameter input forms
- [ ] API trigger (authenticated webhook endpoint)
- [ ] Pipeline-to-pipeline trigger (cascade)
- [ ] Tag-based trigger with semver filtering
- [ ] External event trigger (generic webhook)
- [ ] File-change-based conditional trigger
- [ ] Deployment approval gates (manual approval step)

### 2.3 Pipeline Configuration
- [ ] YAML pipeline definition (`flowforge.yml`)
- [ ] JSON pipeline definition
- [ ] Visual pipeline builder (drag-and-drop in UI)
- [ ] Pipeline versioning (stored in DB and optionally in repo)
- [ ] Pipeline templates library (community + custom)
- [ ] Reusable step components (shared across pipelines)
- [ ] Matrix builds (test across multiple versions/OSes)
- [ ] Conditional step execution (`if` expressions)
- [ ] Step dependencies (`needs` / `depends_on`)
- [ ] Parallel step groups
- [ ] Sequential stage enforcement
- [ ] Continue-on-error per step
- [ ] Retry policy (count, delay, backoff)
- [ ] Timeout per step and per pipeline
- [ ] Pipeline inheritance / extends
- [ ] Dynamic pipeline generation (scripted pipelines)
- [ ] Environment promotion (dev → staging → prod)

### 2.4 Execution Engine
- [ ] Local process executor
- [ ] Docker container executor (image pull, volume mount, network)
- [ ] Kubernetes pod executor (job spec generation)
- [ ] Distributed agent pool (worker nodes register via gRPC)
- [ ] Agent auto-scaling (scale agents on demand)
- [ ] Agent labels and targeting (run on specific agent groups)
- [ ] Workspace isolation per build
- [ ] Shared cache volumes (node_modules, .m2, pip cache)
- [ ] Resource limits (CPU, memory) per step
- [ ] Concurrent pipeline cap (global + per-project)
- [ ] Queue management with priority
- [ ] Dead-letter queue for failed dispatches
- [ ] Build cancellation (graceful + force)
- [ ] Build restart / rerun from failed step
- [ ] Fan-out / fan-in execution pattern

### 2.5 Language & Framework Detection
- [ ] Auto-detect: Go, Node.js, Python, Ruby, Java, Kotlin, Rust, PHP, .NET, Swift
- [ ] Framework detection: React, Next.js, Django, Rails, Spring Boot, Laravel, etc.
- [ ] Dependency file detection: `go.mod`, `package.json`, `requirements.txt`, `Gemfile`, `pom.xml`, `Cargo.toml`, `composer.json`
- [ ] Dockerfile detection → use existing Dockerfile
- [ ] Docker Compose detection
- [ ] Kubernetes manifest detection
- [ ] Terraform / Pulumi detection
- [ ] Auto-generated starter pipeline from detection results
- [ ] Confidence scoring for detected stack

### 2.6 Artifact Management
- [ ] Artifact upload from pipeline steps
- [ ] Artifact download in downstream pipelines (cross-pipeline deps)
- [ ] Artifact expiry policies
- [ ] Artifact storage backends: local disk, S3-compatible (MinIO, AWS S3, GCS, Azure Blob)
- [ ] Build artifact versioning and tagging
- [ ] Docker image registry push integration
- [ ] npm / PyPI / Maven publish steps (built-in actions)
- [ ] Artifact integrity (SHA256 checksums)
- [ ] Artifact browsing UI

### 2.7 Environment & Secrets Management
- [ ] Project-level environment variables
- [ ] Pipeline-level environment variable overrides
- [ ] Step-level environment variable injection
- [ ] Secret store (AES-256-GCM encrypted in SQLite)
- [ ] Secret masking in logs (regex-based scrubbing)
- [ ] External secret provider integration: HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager
- [ ] `.env` file injection per step
- [ ] Shared variable groups across projects
- [ ] Secret rotation notifications
- [ ] Audit log for secret access

### 2.8 Real-Time Monitoring & Logging
- [ ] Live log streaming (WebSocket + SSE fallback)
- [ ] ANSI color code rendering in log viewer
- [ ] Log search and filtering
- [ ] Log retention policies (configurable per project)
- [ ] Log export (download as `.txt` / `.json`)
- [ ] Log forwarding: Loki, Elasticsearch, Splunk, CloudWatch
- [ ] Pipeline run timeline visualization
- [ ] Step-level duration tracking
- [ ] Resource utilization graphs (CPU, memory per step)
- [ ] Build queue depth metrics
- [ ] Agent health dashboard
- [ ] Historical build trends (pass rate, avg duration)
- [ ] Flaky test detection across builds
- [ ] Pipeline SLA tracking

### 2.9 Notifications
- [ ] Slack integration (per-channel, per-event)
- [ ] Email notifications (SMTP + HTML templates)
- [ ] Microsoft Teams webhook
- [ ] Discord webhook
- [ ] PagerDuty integration (on pipeline failure)
- [ ] Generic outbound webhook
- [ ] Notification rules (on: success, failure, change, always)
- [ ] Per-user notification preferences
- [ ] Digest mode (batched notifications)
- [ ] In-app notification center

### 2.10 Security & Access Control
- [ ] RBAC: Owner, Admin, Developer, Viewer roles
- [ ] Project-level role assignment
- [ ] Team / organization management
- [ ] SSO: OIDC / SAML 2.0 / OAuth2 (GitHub, GitLab, Google)
- [ ] API key management (scoped, expiring)
- [ ] IP allowlist for webhook endpoints
- [ ] Audit log (all mutations with actor, timestamp, diff)
- [ ] Two-factor authentication (TOTP)
- [ ] Pipeline run approval workflows
- [ ] Signed commit verification
- [ ] Secret scanning on repo import
- [ ] SBOM generation per build
- [ ] CVE scanning integration (Trivy, Grype)

### 2.11 Docker & Kubernetes Integration
- [ ] Docker-in-Docker (DinD) runner support
- [ ] Docker socket mount option
- [ ] Kubernetes job executor (dynamic pod provisioning)
- [ ] Kubernetes namespace isolation per project
- [ ] Helm chart deployment step
- [ ] kubectl apply step
- [ ] Kubernetes rollout status monitoring
- [ ] Docker build cache (BuildKit)
- [ ] Multi-arch image builds (buildx)
- [ ] Container registry credential management

### 2.12 Testing & Quality Gates
- [ ] JUnit XML test result parsing and display
- [ ] Code coverage threshold gates
- [ ] Test result trending
- [ ] SonarQube integration
- [ ] Codecov / Coveralls integration
- [ ] Lint gate (fail on lint errors)
- [ ] Static analysis integration (gosec, eslint, pylint)
- [ ] Performance regression detection
- [ ] E2E test runner integration (Playwright, Cypress)
- [ ] Load test integration (k6, Locust)

### 2.13 Deployment Management
- [ ] Deployment environments (dev, staging, production)
- [ ] Deployment history and rollback
- [ ] Blue/green deployment support
- [ ] Canary deployment support
- [ ] Environment-specific variable overrides
- [ ] Deployment approval gates
- [ ] Deployment lock (prevent concurrent deploys)
- [ ] Post-deployment smoke tests
- [ ] Deployment diff viewer

### 2.14 API & Integrations
- [ ] REST API (full CRUD for all resources)
- [ ] WebSocket API (real-time events)
- [ ] GraphQL API (optional)
- [ ] gRPC API (agent communication)
- [ ] OpenAPI 3.0 spec (auto-generated)
- [ ] Terraform provider
- [ ] CLI tool (`flowforge` CLI)
- [ ] VS Code extension scaffold
- [ ] Zapier / n8n webhook compatibility

### 2.15 Platform Administration
- [ ] System health dashboard
- [ ] License management
- [ ] Storage quota management
- [ ] Runner / agent management UI
- [ ] Backup and restore
- [ ] Database migration tooling
- [ ] Feature flags
- [ ] Rate limiting (per user, per project)
- [ ] Multi-tenant mode
- [ ] White-labeling support

---

## 3. REPOSITORY STRUCTURE

```
flowforge/
├── CLAUDE.md                        ← This file
├── README.md
├── Makefile
├── docker-compose.yml
├── docker-compose.dev.yml
├── .env.example
│
├── backend/                         ← Go backend
│   ├── cmd/
│   │   ├── server/main.go           ← Main API server
│   │   ├── agent/main.go            ← Worker agent binary
│   │   └── cli/main.go              ← flowforge CLI
│   ├── internal/
│   │   ├── api/                     ← GoFiber route handlers
│   │   │   ├── middleware/
│   │   │   ├── handlers/
│   │   │   └── router.go
│   │   ├── agent/                   ← Agent pool & gRPC server
│   │   │   ├── pool.go
│   │   │   ├── grpc_server.go
│   │   │   └── heartbeat.go
│   │   ├── auth/                    ← JWT, RBAC, SSO
│   │   ├── config/                  ← App config (env + file)
│   │   ├── db/                      ← sqlx + SQLite migrations
│   │   │   ├── migrations/
│   │   │   └── queries/
│   │   ├── detector/                ← Language/framework detector
│   │   ├── engine/                  ← Pipeline execution engine
│   │   │   ├── executor/
│   │   │   │   ├── local.go
│   │   │   │   ├── docker.go
│   │   │   │   └── kubernetes.go
│   │   │   ├── queue/
│   │   │   ├── scheduler/
│   │   │   └── runner.go
│   │   ├── integrations/
│   │   │   ├── github/
│   │   │   ├── gitlab/
│   │   │   ├── bitbucket/
│   │   │   ├── slack/
│   │   │   ├── vault/
│   │   │   └── registry/
│   │   ├── models/                  ← Domain models
│   │   ├── notifications/
│   │   ├── pipeline/                ← YAML/JSON DSL parser
│   │   ├── secrets/                 ← Encrypted secret store
│   │   ├── storage/                 ← Artifact storage backends
│   │   ├── websocket/               ← Real-time log streaming
│   │   └── worker/                  ← Job worker goroutines
│   ├── pkg/                         ← Reusable public packages
│   │   ├── logger/
│   │   ├── crypto/
│   │   ├── events/
│   │   └── utils/
│   ├── proto/                       ← Protobuf definitions
│   │   └── agent.proto
│   ├── go.mod
│   └── go.sum
│
├── frontend/                        ← SolidJS frontend
│   ├── src/
│   │   ├── api/                     ← API client (typed)
│   │   ├── components/
│   │   │   ├── ui/                  ← Base UI components
│   │   │   ├── pipeline/
│   │   │   ├── logs/
│   │   │   ├── agents/
│   │   │   └── settings/
│   │   ├── pages/
│   │   │   ├── dashboard/
│   │   │   ├── projects/
│   │   │   ├── pipelines/
│   │   │   ├── runs/
│   │   │   ├── agents/
│   │   │   ├── settings/
│   │   │   └── auth/
│   │   ├── stores/                  ← SolidJS signals & stores
│   │   ├── hooks/
│   │   ├── types/
│   │   └── App.tsx
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   └── package.json
│
├── agent/                           ← Standalone agent Docker image
│   └── Dockerfile
│
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile.backend
│   │   ├── Dockerfile.frontend
│   │   └── Dockerfile.agent
│   ├── kubernetes/
│   │   ├── namespace.yaml
│   │   ├── deployment-server.yaml
│   │   ├── deployment-agent.yaml
│   │   ├── service.yaml
│   │   ├── ingress.yaml
│   │   ├── pvc.yaml
│   │   └── secrets.yaml
│   ├── helm/
│   │   └── flowforge/
│   ├── terraform/
│   │   ├── aws/
│   │   ├── gcp/
│   │   └── azure/
│   └── ansible/
│
├── docs/
│   ├── api/                         ← OpenAPI spec
│   ├── architecture/
│   ├── guides/
│   └── pipeline-syntax.md
│
└── scripts/
    ├── dev-setup.sh
    ├── seed-db.sh
    └── gen-proto.sh
```

---

## 4. TECH STACK & DEPENDENCIES

### Backend (Go)
```go
// go.mod — key dependencies
require (
    github.com/gofiber/fiber/v3          v3.1.0
    github.com/gofiber/contrib/jwt       latest
    github.com/gofiber/websocket/v2      latest
    github.com/jmoiron/sqlx              latest
    github.com/mattn/go-sqlite3          latest        // CGO_ENABLED=1
    golang.org/x/crypto                  latest
    github.com/golang-jwt/jwt/v5         latest
    github.com/go-playground/validator/v10 latest
    github.com/robfig/cron/v3            latest        // cron scheduler
    github.com/docker/docker             latest        // Docker SDK
    k8s.io/client-go                     latest        // Kubernetes
    google.golang.org/grpc               latest        // agent gRPC
    google.golang.org/protobuf           latest
    github.com/google/go-github/v60      latest
    github.com/xanzy/go-gitlab           latest
    gopkg.in/yaml.v3                     latest
    github.com/minio/minio-go/v7         latest
    github.com/hashicorp/vault/api       latest
    github.com/rs/zerolog                latest
    github.com/spf13/viper               latest
    github.com/pressly/goose/v3          latest        // DB migrations
    github.com/google/uuid               latest
    golang.org/x/sync                    latest        // errgroup
)
```

### Frontend (SolidJS)
```json
{
  "dependencies": {
    "solid-js": "^1.8",
    "@solidjs/router": "^0.13",
    "@solidjs/meta": "^0.29",
    "solid-icons": "^1.1",
    "xterm": "^5.3",
    "xterm-addon-fit": "^0.8",
    "xterm-addon-web-links": "^0.9",
    "monaco-editor": "^0.47",
    "date-fns": "^3.6",
    "chart.js": "^4.4",
    "solid-chartjs": "latest",
    "@codemirror/lang-yaml": "latest",
    "js-yaml": "^4.1"
  },
  "devDependencies": {
    "vite": "^5",
    "vite-plugin-solid": "^2.10",
    "tailwindcss": "^4",
    "@tailwindcss/vite": "^4",
    "typescript": "^5.4"
  }
}
```

---

## 5. ARCHITECTURE OVERVIEW

```
┌─────────────────────────────────────────────────────────────┐
│                        FlowForge                            │
│                                                             │
│  ┌──────────┐    ┌─────────────────────────────────────┐   │
│  │ SolidJS  │◄──►│          GoFiber API Server          │   │
│  │   UI     │    │  REST + WebSocket + SSE              │   │
│  └──────────┘    └──────────┬──────────────────────────┘   │
│                             │                               │
│          ┌──────────────────┼───────────────────┐          │
│          │                  │                   │          │
│   ┌──────▼──────┐  ┌───────▼──────┐  ┌────────▼────────┐  │
│   │  Pipeline   │  │    Secret    │  │   Notification  │  │
│   │   Engine    │  │    Store     │  │    Dispatcher   │  │
│   └──────┬──────┘  └─────────────┘  └─────────────────┘  │
│          │                                                  │
│   ┌──────▼──────────────────────────────────┐              │
│   │              Job Queue                   │              │
│   │    (in-memory + SQLite persistence)      │              │
│   └──────┬──────────────────────────────────┘              │
│          │                                                  │
│   ┌──────┴────────────────────────────────────┐            │
│   │           Agent Pool (gRPC)                │            │
│   │  ┌──────────┐ ┌──────────┐ ┌──────────┐   │            │
│   │  │ Agent 1  │ │ Agent 2  │ │ Agent N  │   │            │
│   │  │ (local)  │ │ (docker) │ │  (k8s)   │   │            │
│   │  └──────────┘ └──────────┘ └──────────┘   │            │
│   └───────────────────────────────────────────┘            │
│                                                             │
│   ┌─────────────────────────────────────────┐              │
│   │        SQLite (WAL mode) via sqlx         │              │
│   └─────────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

### Key Architectural Decisions
1. **SQLite WAL mode** — Single writer, concurrent readers; perfect for small-to-medium deployments. Migrations via `goose`.
2. **GoFiber v3** — High-performance HTTP using fasthttp; middleware chain for auth, logging, rate limiting.
3. **gRPC agent protocol** — Agents register, receive job assignments, stream logs back bidirectionally.
4. **WebSocket log streaming** — Each pipeline run has a dedicated WebSocket channel; logs are buffered in SQLite for replay.
5. **Event bus** — Internal Go channel-based event bus for decoupled notification dispatch.
6. **SolidJS fine-grained reactivity** — Signals for real-time log updates without full re-renders.

---

## 6. SUBAGENT TASK BREAKDOWN (PARALLEL WORKSTREAMS)

Run these workstreams in parallel for maximum velocity. Each subagent should work in isolation and produce complete, tested code.

### Workstream A — Backend Core Infrastructure
**Files:** `backend/cmd/server/`, `backend/internal/config/`, `backend/internal/db/`, `backend/internal/api/middleware/`, `backend/pkg/logger/`
**Tasks:**
1. Bootstrap GoFiber v3 app with graceful shutdown
2. SQLite + WAL config via sqlx; connection pool settings
3. Goose migration runner on startup
4. Zerolog structured logging with request ID middleware
5. Viper config loading (env vars + `config.yaml`)
6. JWT middleware (access + refresh tokens)
7. Rate limiter middleware (per-IP + per-user)
8. CORS + Helmet security headers
9. Request validation middleware using go-playground/validator
10. OpenAPI spec generator middleware

### Workstream B — Database Schema & Migrations
**Files:** `backend/internal/db/migrations/`, `backend/internal/db/queries/`, `backend/internal/models/`
**Tasks:** Implement all migrations (see Section 9). Write sqlx query functions for every model. Implement soft-delete pattern. Write repository interfaces.

### Workstream C — Authentication & RBAC
**Files:** `backend/internal/auth/`
**Tasks:**
1. Local auth (bcrypt password hashing)
2. OAuth2: GitHub, GitLab, Google
3. OIDC provider support
4. TOTP (2FA) using `github.com/pquerna/otp`
5. RBAC middleware — Owner / Admin / Developer / Viewer
6. API key generation (scoped, expiring, hashed storage)
7. Session management
8. Audit log writer

### Workstream D — Repository Integrations
**Files:** `backend/internal/integrations/github/`, `gitlab/`, `bitbucket/`
**Tasks:**
1. GitHub App installation + OAuth flow
2. GitHub webhook receiver (HMAC-SHA256 validation)
3. Event parsing: push, pull_request, create (tag), release, workflow_dispatch
4. GitLab webhook receiver + token validation
5. Bitbucket webhook receiver
6. Repository clone helper (shallow clone, SSH/HTTPS)
7. Branch list, commit info, file tree APIs
8. Status/check reporting back to SCM

### Workstream E — Pipeline DSL Parser
**Files:** `backend/internal/pipeline/`
**Tasks:**
1. Define Go structs for pipeline spec (stages, jobs, steps, matrix, conditions)
2. YAML parser with strict validation
3. JSON parser
4. Expression evaluator for `if` conditions
5. Matrix expansion engine
6. Pipeline validator (dependency graph, cycle detection)
7. Template resolver (inline + referenced templates)
8. Dynamic pipeline generator (scripted pipelines)
9. Pipeline diff engine (for versioning)

### Workstream F — Execution Engine
**Files:** `backend/internal/engine/`
**Tasks:**
1. Job queue (priority queue + SQLite persistence)
2. Scheduler goroutine (picks queued jobs, dispatches to agents)
3. Local executor (os/exec with pty for color logs)
4. Docker executor (Docker SDK: create, start, attach, remove)
5. Kubernetes executor (batch/v1 Job, watch until complete)
6. Workspace manager (temp dirs, cleanup)
7. Cache manager (tar/untar cache volumes)
8. Artifact collector (post-step artifact upload)
9. Log streamer (goroutine → WebSocket broadcast)
10. Retry/timeout manager
11. Build cancellation (context cancellation propagation)
12. Fan-out/fan-in parallel step coordinator

### Workstream G — Agent System
**Files:** `backend/internal/agent/`, `backend/cmd/agent/`, `backend/proto/`
**Tasks:**
1. Define `agent.proto` (Register, Heartbeat, AssignJob, StreamLogs, ReportStatus)
2. Generate gRPC code
3. gRPC server (accepts agent connections)
4. Agent pool manager (track capacity, health, labels)
5. Job dispatcher (match job requirements to agent capabilities)
6. Agent binary (connects to server, executes jobs, streams logs)
7. Agent auto-scaler interface
8. Agent health monitoring + eviction

### Workstream H — Language/Framework Detector
**Files:** `backend/internal/detector/`
**Tasks:**
1. File tree walker
2. Language detectors (one per language using indicator files + content heuristics)
3. Framework detector (package.json scripts, spring annotations, etc.)
4. Dependency extractor per ecosystem
5. Confidence scorer
6. Pipeline auto-generator from detection results
7. Dockerfile analyzer
8. Docker Compose analyzer

### Workstream I — Secrets & Security
**Files:** `backend/internal/secrets/`, `backend/pkg/crypto/`
**Tasks:**
1. AES-256-GCM encryption/decryption helpers
2. Encrypted secret store (CRUD with sqlx)
3. Secret injection into step environment
4. Log scrubber (replace secret values with `***`)
5. HashiCorp Vault integration
6. AWS Secrets Manager integration
7. GCP Secret Manager integration
8. Secret rotation event hooks

### Workstream J — Notifications
**Files:** `backend/internal/notifications/`
**Tasks:**
1. Internal event bus (typed Go channels)
2. Slack notifier (Block Kit messages with pipeline summary)
3. Email notifier (SMTP + HTML template with Go `html/template`)
4. Microsoft Teams notifier (Adaptive Cards)
5. Discord notifier
6. PagerDuty notifier
7. Generic outbound webhook (retry with exponential backoff)
8. Notification rule engine (evaluate conditions per event)
9. User notification preferences store
10. In-app notification store + WebSocket push

### Workstream K — Artifact Storage
**Files:** `backend/internal/storage/`
**Tasks:**
1. Storage backend interface
2. Local disk backend
3. S3-compatible backend (MinIO / AWS S3 / GCS / Azure Blob via minio-go)
4. Artifact metadata in SQLite
5. Upload/download handlers in GoFiber (multipart)
6. Artifact expiry worker (background goroutine)
7. Cross-pipeline artifact dependency resolver
8. Checksum verification

### Workstream L — WebSocket & Real-Time
**Files:** `backend/internal/websocket/`
**Tasks:**
1. GoFiber WebSocket upgrade handler
2. Hub pattern (rooms per pipeline run)
3. Log broadcast goroutine
4. Event broadcast (pipeline status changes)
5. SSE fallback endpoint
6. Connection authentication (JWT in query param)
7. Log replay on reconnect (fetch from SQLite, then subscribe live)

### Workstream M — REST API Handlers
**Files:** `backend/internal/api/handlers/`
**Tasks:** Implement handlers for all resources. See Section 13 for full route list. Each handler must: authenticate, authorize (RBAC), validate input, call service layer, return typed JSON response.

### Workstream N — Frontend Foundation
**Files:** `frontend/src/`
**Tasks:**
1. Vite + SolidJS + TailwindCSS v4 project scaffold
2. SolidJS Router setup (all routes)
3. Auth store (JWT storage, refresh logic)
4. API client (typed fetch wrapper with error handling)
5. WebSocket client (auto-reconnect, event dispatching)
6. Base UI components: Button, Input, Select, Modal, Toast, Badge, Dropdown, Table, Tabs, Card
7. Layout: Sidebar, TopBar, Breadcrumb, PageContainer
8. Dark mode support

### Workstream O — Frontend Features
**Files:** `frontend/src/pages/`, `frontend/src/components/`
**Tasks:**
1. Dashboard: pipeline run stats, recent runs, agent status
2. Project list + create project page
3. Repository connection wizard (GitHub/GitLab/Bitbucket OAuth flow)
4. Pipeline list + create/edit pipeline (Monaco YAML editor)
5. Visual pipeline builder (drag-and-drop DAG editor)
6. Pipeline run detail page (stage/step tree + log viewer)
7. Real-time log viewer (xterm.js with ANSI support)
8. Agent management page
9. Secret management page
10. Notification settings page
11. User management + RBAC page
12. API keys page
13. Audit log viewer
14. System settings page
15. Artifact browser

### Workstream P — Deployment & Infrastructure
**Files:** `deploy/`
**Tasks:**
1. `Dockerfile.backend` (multi-stage, CGO for sqlite3)
2. `Dockerfile.frontend` (build + nginx serve)
3. `Dockerfile.agent`
4. `docker-compose.yml` (full stack)
5. Kubernetes manifests (all resources)
6. Helm chart
7. Terraform modules: AWS (EKS + RDS Aurora + S3), GCP (GKE + Cloud SQL + GCS), Azure (AKS + Azure SQL + Blob)
8. GitHub Actions workflow for CI of FlowForge itself

### Workstream Q — CLI Tool
**Files:** `backend/cmd/cli/`
**Tasks:**
1. cobra CLI scaffold
2. `flowforge login` (API key or browser OAuth)
3. `flowforge run <pipeline>` (trigger + stream logs)
4. `flowforge pipelines list/get/create/update/delete`
5. `flowforge agents list`
6. `flowforge secrets set/delete/list`
7. `flowforge artifacts download`
8. Config file (`~/.flowforge/config.yaml`)

---

## 7. BACKEND IMPLEMENTATION GUIDE

### GoFiber v3 App Bootstrap (`backend/cmd/server/main.go`)
```go
package main

import (
    "context"
    "os/signal"
    "syscall"

    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/compress"
    "github.com/gofiber/fiber/v3/middleware/cors"
    "github.com/gofiber/fiber/v3/middleware/helmet"
    "github.com/gofiber/fiber/v3/middleware/limiter"
    "github.com/gofiber/fiber/v3/middleware/logger"
    "github.com/gofiber/fiber/v3/middleware/recover"

    "flowforge/internal/api"
    "flowforge/internal/config"
    "flowforge/internal/db"
    "flowforge/internal/engine"
    "flowforge/internal/websocket"
)

func main() {
    cfg := config.Load()
    database := db.Connect(cfg.DatabasePath) // WAL mode
    db.Migrate(database)

    hub := websocket.NewHub()
    go hub.Run()

    eng := engine.New(database, hub, cfg)
    go eng.Start()

    app := fiber.New(fiber.Config{
        AppName:      "FlowForge v1.0",
        ErrorHandler: api.ErrorHandler,
        BodyLimit:    50 * 1024 * 1024, // 50MB
    })

    // Global middleware
    app.Use(recover.New())
    app.Use(logger.New())
    app.Use(helmet.New())
    app.Use(cors.New(cors.Config{AllowOrigins: cfg.AllowedOrigins}))
    app.Use(compress.New())
    app.Use(limiter.New(limiter.Config{Max: 100, Expiration: 60}))

    // Register routes
    api.RegisterRoutes(app, database, eng, hub, cfg)

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go app.Listen(":" + cfg.Port)
    <-ctx.Done()
    app.Shutdown()
}
```

### SQLite WAL Setup (`backend/internal/db/db.go`)
```go
package db

import (
    "fmt"
    "github.com/jmoiron/sqlx"
    _ "github.com/mattn/go-sqlite3"
)

func Connect(path string) *sqlx.DB {
    dsn := fmt.Sprintf(
        "file:%s?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=on&_busy_timeout=5000&cache=shared",
        path,
    )
    db := sqlx.MustConnect("sqlite3", dsn)
    db.SetMaxOpenConns(1)        // SQLite: single writer
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)
    return db
}
```

### Pipeline Execution Flow
```
Webhook received
    → Validate signature
    → Parse event (push/PR/tag)
    → Match trigger rules
    → Resolve pipeline config (repo file OR DB stored)
    → Create PipelineRun record (status: queued)
    → Enqueue job with priority
    → Scheduler picks job
    → Select available agent (matching labels)
    → Dispatch via gRPC (or run locally)
    → Agent clones repo (shallow)
    → Detect language if auto-config
    → Execute stages sequentially
        → Execute jobs in stage (parallel if no deps)
            → Execute steps sequentially
                → Inject env vars + secrets
                → Stream logs → WebSocket hub → frontend
                → Collect artifacts
    → Report final status
    → Update PipelineRun record
    → Trigger notifications
    → Trigger downstream pipelines (if configured)
```

---

## 8. FRONTEND IMPLEMENTATION GUIDE

### Key SolidJS Patterns

```typescript
// Real-time log store using SolidJS signals
import { createSignal, createStore } from "solid-js";

const [logs, setLogs] = createStore<LogLine[]>([]);
const ws = new WebSocket(`/ws/runs/${runId}/logs`);

ws.onmessage = (e) => {
  const line: LogLine = JSON.parse(e.data);
  setLogs(logs.length, line); // Fine-grained update
};

// xterm.js integration
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";

const term = new Terminal({
  convertEol: true,
  theme: { background: "#0d1117" }
});
const fitAddon = new FitAddon();
term.loadAddon(fitAddon);
term.open(containerRef);
fitAddon.fit();

// Write logs as they arrive
ws.onmessage = (e) => term.write(e.data);
```

### Route Structure
```typescript
// src/App.tsx
import { Router, Route } from "@solidjs/router";

const App = () => (
  <Router>
    <Route path="/auth/*" component={AuthLayout}>
      <Route path="/login" component={LoginPage} />
      <Route path="/oauth/callback" component={OAuthCallback} />
    </Route>
    <Route path="/" component={AppLayout} middleware={[requireAuth]}>
      <Route path="/" component={DashboardPage} />
      <Route path="/projects" component={ProjectsPage} />
      <Route path="/projects/:id" component={ProjectDetailPage} />
      <Route path="/projects/:id/pipelines" component={PipelinesPage} />
      <Route path="/projects/:id/pipelines/:pid" component={PipelineDetailPage} />
      <Route path="/projects/:id/pipelines/:pid/runs/:rid" component={RunDetailPage} />
      <Route path="/agents" component={AgentsPage} />
      <Route path="/settings/*" component={SettingsLayout} />
      <Route path="/admin/*" component={AdminLayout} middleware={[requireAdmin]} />
    </Route>
  </Router>
);
```

---

## 9. DATABASE SCHEMA

### Core Migrations (SQLite WAL)

```sql
-- 001_create_users.sql
CREATE TABLE users (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    email       TEXT NOT NULL UNIQUE,
    username    TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    display_name TEXT,
    avatar_url  TEXT,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK(role IN ('owner','admin','developer','viewer')),
    totp_secret TEXT,
    totp_enabled INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

-- 002_create_organizations.sql
CREATE TABLE organizations (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    logo_url    TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE org_members (
    org_id      TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'developer',
    joined_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (org_id, user_id)
);

-- 003_create_projects.sql
CREATE TABLE projects (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    org_id      TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT,
    visibility  TEXT NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','internal','public')),
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME,
    UNIQUE(org_id, slug)
);

-- 004_create_repositories.sql
CREATE TABLE repositories (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider        TEXT NOT NULL CHECK(provider IN ('github','gitlab','bitbucket')),
    provider_id     TEXT NOT NULL,
    full_name       TEXT NOT NULL,
    clone_url       TEXT NOT NULL,
    ssh_url         TEXT,
    default_branch  TEXT NOT NULL DEFAULT 'main',
    webhook_id      TEXT,
    webhook_secret  TEXT,
    access_token_enc TEXT,
    ssh_key_enc     TEXT,
    is_active       INTEGER NOT NULL DEFAULT 1,
    last_sync_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 005_create_pipelines.sql
CREATE TABLE pipelines (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repository_id   TEXT REFERENCES repositories(id),
    name            TEXT NOT NULL,
    description     TEXT,
    config_source   TEXT NOT NULL DEFAULT 'db' CHECK(config_source IN ('db','repo')),
    config_path     TEXT DEFAULT '.flowforge.yml',
    config_content  TEXT,
    config_version  INTEGER NOT NULL DEFAULT 1,
    triggers        TEXT NOT NULL DEFAULT '{}',  -- JSON
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME
);

CREATE TABLE pipeline_versions (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    pipeline_id TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    config      TEXT NOT NULL,
    message     TEXT,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, version)
);

-- 006_create_pipeline_runs.sql
CREATE TABLE pipeline_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    pipeline_id     TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    number          INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK(status IN ('queued','pending','running','success','failure','cancelled','skipped','waiting_approval')),
    trigger_type    TEXT NOT NULL CHECK(trigger_type IN ('push','pull_request','schedule','manual','api','pipeline')),
    trigger_data    TEXT,       -- JSON
    commit_sha      TEXT,
    commit_message  TEXT,
    branch          TEXT,
    tag             TEXT,
    author          TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER,
    created_by      TEXT REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pipeline_id, number)
);

CREATE TABLE stage_runs (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    position    INTEGER NOT NULL,
    started_at  DATETIME,
    finished_at DATETIME
);

CREATE TABLE job_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    stage_run_id    TEXT NOT NULL REFERENCES stage_runs(id) ON DELETE CASCADE,
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    agent_id        TEXT,
    executor_type   TEXT NOT NULL DEFAULT 'local',
    started_at      DATETIME,
    finished_at     DATETIME
);

CREATE TABLE step_runs (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    job_run_id      TEXT NOT NULL REFERENCES job_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    exit_code       INTEGER,
    error_message   TEXT,
    started_at      DATETIME,
    finished_at     DATETIME,
    duration_ms     INTEGER
);

-- 007_create_logs.sql
CREATE TABLE run_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step_run_id TEXT REFERENCES step_runs(id) ON DELETE CASCADE,
    stream      TEXT NOT NULL DEFAULT 'stdout' CHECK(stream IN ('stdout','stderr','system')),
    content     TEXT NOT NULL,
    ts          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_run_logs_run_id ON run_logs(run_id);
CREATE INDEX idx_run_logs_step_id ON run_logs(step_run_id);

-- 008_create_agents.sql
CREATE TABLE agents (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    labels      TEXT NOT NULL DEFAULT '{}',  -- JSON array
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

-- 009_create_secrets.sql
CREATE TABLE secrets (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    org_id      TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL CHECK(scope IN ('project','org','global')),
    key         TEXT NOT NULL,
    value_enc   TEXT NOT NULL,  -- AES-256-GCM encrypted
    masked      INTEGER NOT NULL DEFAULT 1,
    created_by  TEXT REFERENCES users(id),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 010_create_artifacts.sql
CREATE TABLE artifacts (
    id              TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    run_id          TEXT NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    step_run_id     TEXT REFERENCES step_runs(id),
    name            TEXT NOT NULL,
    path            TEXT NOT NULL,
    size_bytes      INTEGER,
    checksum_sha256 TEXT,
    storage_backend TEXT NOT NULL DEFAULT 'local',
    storage_key     TEXT NOT NULL,
    expire_at       DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 011_create_notifications.sql
CREATE TABLE notification_channels (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    project_id  TEXT REFERENCES projects(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK(type IN ('slack','email','teams','discord','pagerduty','webhook')),
    name        TEXT NOT NULL,
    config_enc  TEXT NOT NULL,  -- encrypted config JSON
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 012_create_audit_log.sql
CREATE TABLE audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_id    TEXT REFERENCES users(id),
    actor_ip    TEXT,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT,
    changes     TEXT,   -- JSON diff
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);
```

---

## 10. PIPELINE CONFIGURATION DSL

### Full `flowforge.yml` Syntax

```yaml
version: "1"
name: "My App CI/CD"

# Trigger configuration
on:
  push:
    branches: ["main", "develop", "feature/**"]
    paths: ["src/**", "go.mod"]
    ignore_paths: ["docs/**", "*.md"]
  pull_request:
    types: [opened, synchronize, reopened]
    branches: ["main"]
  schedule:
    - cron: "0 2 * * *"
      timezone: "Asia/Kathmandu"
      branch: "main"
  manual:
    inputs:
      environment:
        description: "Deploy target"
        required: true
        default: "staging"
        type: choice
        options: [staging, production]

# Global defaults
defaults:
  timeout: 30m
  retry:
    count: 2
    delay: 30s
    on: [failure]
  executor: docker
  image: ubuntu:22.04

# Environment variables (non-secret)
env:
  APP_NAME: myapp
  GO_VERSION: "1.22"

# Stages define execution order
stages:
  - setup
  - test
  - build
  - security
  - publish
  - deploy

# Jobs
jobs:
  detect:
    stage: setup
    steps:
      - name: Detect stack
        uses: flowforge/detect-stack@v1
        outputs:
          - DETECTED_LANGUAGE
          - DETECTED_FRAMEWORK

  test-unit:
    stage: test
    executor: docker
    image: golang:${{ env.GO_VERSION }}-alpine
    env:
      CGO_ENABLED: "0"
    cache:
      - key: go-mod-${{ hash("go.sum") }}
        paths: [/go/pkg/mod]
    steps:
      - name: Checkout
        uses: flowforge/checkout@v1
      - name: Run tests
        run: go test ./... -coverprofile=coverage.out
      - name: Upload coverage
        uses: flowforge/upload-artifact@v1
        with:
          name: coverage-report
          path: coverage.out

  test-matrix:
    stage: test
    matrix:
      go_version: ["1.21", "1.22"]
      os: [ubuntu-22.04, alpine]
    executor: docker
    image: golang:${{ matrix.go_version }}
    steps:
      - uses: flowforge/checkout@v1
      - run: go test ./...

  build:
    stage: build
    needs: [test-unit]
    executor: docker
    image: golang:${{ env.GO_VERSION }}-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Build binary
        run: |
          CGO_ENABLED=1 go build -ldflags="-w -s" -o dist/server ./cmd/server
      - uses: flowforge/upload-artifact@v1
        with:
          name: server-binary
          path: dist/server
          retention_days: 30

  docker-build:
    stage: publish
    needs: [build]
    executor: docker
    image: docker:26-dind
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Download binary
        uses: flowforge/download-artifact@v1
        with:
          name: server-binary
          path: dist/
      - name: Build and push image
        uses: flowforge/docker-build-push@v1
        with:
          registry: ghcr.io
          image: ${{ secrets.REGISTRY_IMAGE }}
          tags: |
            latest
            ${{ git.sha }}
          push: ${{ git.branch == 'main' }}

  deploy-staging:
    stage: deploy
    needs: [docker-build]
    environment: staging
    when: ${{ git.branch == 'main' }}
    steps:
      - name: Deploy to Kubernetes
        uses: flowforge/helm-deploy@v1
        with:
          chart: ./deploy/helm/flowforge
          release: flowforge-staging
          namespace: staging
          values: |
            image.tag: ${{ git.sha }}
          kubeconfig: ${{ secrets.STAGING_KUBECONFIG }}

  deploy-prod:
    stage: deploy
    needs: [deploy-staging]
    environment: production
    approval_required: true
    when: ${{ git.tag =~ /^v\d+/ }}
    steps:
      - uses: flowforge/helm-deploy@v1
        with:
          release: flowforge-production
          namespace: production
          kubeconfig: ${{ secrets.PROD_KUBECONFIG }}

# Notification rules
notify:
  on_failure:
    - channel: slack-alerts
    - channel: email-devteam
  on_success:
    when: ${{ was_failing }}
    - channel: slack-alerts
  on_deployment:
    - channel: slack-deployments
```

---

## 11. SECURITY IMPLEMENTATION

### Secret Encryption
```go
// pkg/crypto/aes.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(key []byte, plaintext string) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(sealed), nil
}

func Decrypt(key []byte, ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    nonce, data := data[:nonceSize], data[nonceSize:]
    plain, err := gcm.Open(nil, nonce, data, nil)
    if err != nil {
        return "", err
    }
    return string(plain), nil
}
```

### Log Scrubber
```go
// internal/secrets/scrubber.go
type Scrubber struct {
    patterns []*regexp.Regexp
}

func NewScrubber(secrets []string) *Scrubber {
    s := &Scrubber{}
    for _, secret := range secrets {
        if len(secret) < 4 { continue }
        s.patterns = append(s.patterns, regexp.MustCompile(regexp.QuoteMeta(secret)))
    }
    return s
}

func (s *Scrubber) Scrub(line string) string {
    for _, p := range s.patterns {
        line = p.ReplaceAllString(line, "***")
    }
    return line
}
```

---

## 12. DISTRIBUTED EXECUTION ENGINE

### Agent gRPC Protocol (`proto/agent.proto`)
```protobuf
syntax = "proto3";
package agent;
option go_package = "flowforge/proto/agent";

service AgentService {
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Heartbeat(stream HeartbeatRequest) returns (stream HeartbeatResponse);
    rpc AssignJob(AssignJobRequest) returns (AssignJobResponse);
    rpc ExecuteJob(ExecuteJobRequest) returns (stream JobEvent);
    rpc ReportStatus(ReportStatusRequest) returns (ReportStatusResponse);
}

message RegisterRequest {
    string token = 1;
    string name = 2;
    repeated string labels = 3;
    string executor = 4;
    string os = 5;
    string arch = 6;
    int32 cpu_cores = 7;
    int64 memory_mb = 8;
    string version = 9;
}

message ExecuteJobRequest {
    string job_run_id = 1;
    string config_json = 2;
    map<string, string> env_vars = 3;
    repeated string secrets = 4; // encrypted
    repeated ArtifactRef artifacts = 5;
}

message JobEvent {
    oneof event {
        LogLine log = 1;
        StepStatus step_status = 2;
        JobComplete complete = 3;
    }
}

message LogLine {
    string step_run_id = 1;
    string stream = 2; // stdout|stderr|system
    string content = 3;
    int64 timestamp_ms = 4;
}

message StepStatus {
    string step_run_id = 1;
    string status = 2;
    int32 exit_code = 3;
}

message JobComplete {
    string status = 2;
    int64 duration_ms = 3;
}
```

---

## 13. API REFERENCE STRUCTURE

### Route Groups

```
POST   /api/v1/auth/login
POST   /api/v1/auth/register
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/oauth/:provider
GET    /api/v1/auth/oauth/:provider/callback
POST   /api/v1/auth/totp/setup
POST   /api/v1/auth/totp/verify

GET    /api/v1/users/me
PUT    /api/v1/users/me
GET    /api/v1/users/:id

GET    /api/v1/orgs
POST   /api/v1/orgs
GET    /api/v1/orgs/:id
PUT    /api/v1/orgs/:id
DELETE /api/v1/orgs/:id
GET    /api/v1/orgs/:id/members
POST   /api/v1/orgs/:id/members
DELETE /api/v1/orgs/:id/members/:userId

GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/:id
PUT    /api/v1/projects/:id
DELETE /api/v1/projects/:id

GET    /api/v1/projects/:id/repositories
POST   /api/v1/projects/:id/repositories
DELETE /api/v1/projects/:id/repositories/:repoId
POST   /api/v1/projects/:id/repositories/:repoId/sync

GET    /api/v1/projects/:id/pipelines
POST   /api/v1/projects/:id/pipelines
GET    /api/v1/projects/:id/pipelines/:pid
PUT    /api/v1/projects/:id/pipelines/:pid
DELETE /api/v1/projects/:id/pipelines/:pid
GET    /api/v1/projects/:id/pipelines/:pid/versions
POST   /api/v1/projects/:id/pipelines/:pid/trigger
POST   /api/v1/projects/:id/pipelines/:pid/validate

GET    /api/v1/projects/:id/pipelines/:pid/runs
GET    /api/v1/projects/:id/pipelines/:pid/runs/:rid
POST   /api/v1/projects/:id/pipelines/:pid/runs/:rid/cancel
POST   /api/v1/projects/:id/pipelines/:pid/runs/:rid/rerun
POST   /api/v1/projects/:id/pipelines/:pid/runs/:rid/approve
GET    /api/v1/projects/:id/pipelines/:pid/runs/:rid/logs
GET    /api/v1/projects/:id/pipelines/:pid/runs/:rid/artifacts

GET    /api/v1/projects/:id/secrets
POST   /api/v1/projects/:id/secrets
PUT    /api/v1/projects/:id/secrets/:secretId
DELETE /api/v1/projects/:id/secrets/:secretId

GET    /api/v1/projects/:id/notifications
POST   /api/v1/projects/:id/notifications
PUT    /api/v1/projects/:id/notifications/:nid
DELETE /api/v1/projects/:id/notifications/:nid

GET    /api/v1/agents
POST   /api/v1/agents
GET    /api/v1/agents/:id
DELETE /api/v1/agents/:id
POST   /api/v1/agents/:id/drain

GET    /api/v1/artifacts/:id
GET    /api/v1/artifacts/:id/download

GET    /api/v1/audit-logs

GET    /api/v1/system/health
GET    /api/v1/system/metrics
GET    /api/v1/system/info

# Webhooks (unauthenticated, signature-validated)
POST   /webhooks/github
POST   /webhooks/gitlab
POST   /webhooks/bitbucket

# WebSocket
GET    /ws/runs/:runId/logs
GET    /ws/events
```

---

## 14. DEPLOYMENT & INFRASTRUCTURE

### Docker Compose (Development)
```yaml
# docker-compose.dev.yml
version: "3.9"
services:
  backend:
    build:
      context: ./backend
      dockerfile: ../deploy/docker/Dockerfile.backend
    ports: ["8081:8081"]
    volumes:
      - ./data:/data
      - ./backend:/app  # hot reload with air
    environment:
      - DATABASE_PATH=/data/flowforge.db
      - JWT_SECRET=dev-secret-change-in-prod
      - ENCRYPTION_KEY=0000000000000000000000000000000000000000000000000000000000000000
    command: air -c .air.toml

  frontend:
    build:
      context: ./frontend
    ports: ["3000:3000"]
    volumes:
      - ./frontend/src:/app/src
    environment:
      - VITE_API_URL=http://localhost:8081

  minio:
    image: minio/minio
    ports: ["9000:9000", "9001:9001"]
    volumes: ["./data/minio:/data"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
```

### Kubernetes Deployment
```yaml
# deploy/kubernetes/deployment-server.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flowforge-server
  namespace: flowforge
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flowforge-server
  template:
    spec:
      containers:
        - name: server
          image: flowforge/server:latest
          ports: [{containerPort: 8081}]
          env:
            - name: DATABASE_PATH
              value: /data/flowforge.db
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: flowforge-secrets
                  key: jwt-secret
          volumeMounts:
            - name: data
              mountPath: /data
          resources:
            requests: {cpu: "100m", memory: "128Mi"}
            limits: {cpu: "500m", memory: "512Mi"}
          readinessProbe:
            httpGet: {path: /api/v1/system/health, port: 8081}
          livenessProbe:
            httpGet: {path: /api/v1/system/health, port: 8081}
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: flowforge-data
```

---

## 15. TESTING STRATEGY

```
backend/
  internal/
    api/handlers/    → Integration tests (httptest + fiber.Test)
    engine/          → Unit tests (mock executor)
    pipeline/        → Unit tests (parser, validator)
    detector/        → Unit tests (fixture repos)
    secrets/         → Unit tests (encrypt/decrypt round-trip)
    auth/            → Unit tests + integration

frontend/
  src/
    components/      → vitest + solid-testing-library
    pages/           → E2E with Playwright
```

### Test Commands
```bash
# Backend
cd backend && go test ./... -race -cover

# Frontend unit
cd frontend && pnpm test

# E2E
cd frontend && pnpm playwright test

# Integration (full stack via docker-compose)
make test-integration
```

---

## 16. PROMPTS FOR SUBAGENTS

Use these prompts with Claude Code (`claude`) to spawn parallel workstreams. Run each in a separate terminal.

### Prompt: Workstream A (Backend Core)
```
You are working on FlowForge, a CI/CD platform. Read CLAUDE.md for full context.

Your task: Implement Workstream A — Backend Core Infrastructure.

Files to create:
- backend/cmd/server/main.go
- backend/internal/config/config.go
- backend/internal/db/db.go
- backend/internal/api/router.go
- backend/internal/api/middleware/ (auth, logger, rbac, ratelimit, error)
- backend/pkg/logger/logger.go
- backend/go.mod (with all deps from CLAUDE.md Section 4)

Requirements:
- GoFiber v3.1.0 with all middleware from CLAUDE.md Section 7
- SQLite WAL mode via jmoiron/sqlx exactly as shown in Section 7
- Graceful shutdown with signal handling
- Structured logging with zerolog
- Config loading from env vars + config.yaml via viper
- JWT middleware for protected routes
- Full error handler with typed error responses
- Make all tests pass: go test ./...
```

### Prompt: Workstream B (Database)
```
You are working on FlowForge CI/CD platform. Read CLAUDE.md Section 9 for full schema.

Your task: Implement Workstream B — Database Schema & Migrations.

Files to create:
- backend/internal/db/migrations/ (all 12 migration files from Section 9)
- backend/internal/db/queries/ (sqlx query functions for all models)
- backend/internal/models/ (Go structs for all DB tables, with db/json tags)

Requirements:
- Use goose for migration management
- Implement repository pattern: one file per model with CRUD + list + soft-delete
- Use sqlx.NamedExec for inserts/updates
- Add indexes for all foreign keys and commonly filtered columns
- Write table-driven tests for each repository
```

### Prompt: Workstream D (Repository Integrations)
```
You are working on FlowForge CI/CD platform. Read CLAUDE.md for context.

Your task: Implement Workstream D — Repository Integrations.

Files to create:
- backend/internal/integrations/github/
- backend/internal/integrations/gitlab/
- backend/internal/integrations/bitbucket/

For each provider implement:
1. OAuth2 flow (authorization URL + callback handler)
2. Webhook receiver (signature validation)
3. Event parser (push, PR, tag events → internal TriggerEvent struct)
4. API client (list repos, get commit info, create status checks)
5. Repository clone helper

Use go-github/v60 for GitHub, go-gitlab for GitLab. Use net/http for Bitbucket.
Write unit tests with mock HTTP servers.
```

### Prompt: Workstream F (Execution Engine)
```
You are working on FlowForge CI/CD platform. Read CLAUDE.md Sections 5, 10, 12 carefully.

Your task: Implement Workstream F — Execution Engine.

Files to create:
- backend/internal/engine/ (full directory)

Implement:
1. In-memory priority queue with SQLite persistence for durability
2. Scheduler goroutine (poll queue, dispatch to available agents)
3. Local executor (os/exec with PTY, capture stdout/stderr)
4. Docker executor (docker SDK: pull, create, start, attach, wait, remove)
5. Log streamer (goroutine that broadcasts to WebSocket hub)
6. Artifact collector
7. Retry/timeout/cancellation manager

Use context.Context for cancellation propagation throughout.
The engine must support concurrent execution of multiple jobs.
Write integration tests using testcontainers-go for Docker executor.
```

### Prompt: Workstream N+O (Frontend)
```
You are working on FlowForge CI/CD platform. Read CLAUDE.md Sections 8, 13 for context.

Your task: Implement the complete SolidJS frontend.

Tech: SolidJS 1.8, TailwindCSS v4, TypeScript, Vite 5, xterm.js, Monaco Editor.

Create ALL pages from CLAUDE.md Section 6 Workstream O:
- Full routing setup
- Base UI component library (Button, Input, Select, Modal, Toast, Badge, Table, Tabs, Card)
- Dark mode with CSS variables
- Typed API client matching all routes in Section 13
- WebSocket client with auto-reconnect
- Real-time log viewer using xterm.js
- Pipeline YAML editor using Monaco with schema validation
- Pipeline run detail with stage/step tree
- Agent management page
- Secrets management page

Design: Dark theme, professional DevOps aesthetic (inspired by GitHub Actions + Vercel).
Use TailwindCSS v4 utility classes. No component libraries except what's listed in package.json.
```

---

## QUICK START FOR CLAUDE CODE

```bash
# 1. Initialize the project
mkdir flowforge && cd flowforge
git init

# 2. Start Claude Code
claude

# 3. First prompt to Claude Code:
# "Read CLAUDE.md and begin implementing the project.
#  Start with Workstream A and B in parallel using separate Task tools.
#  Use the exact dependency versions specified.
#  After core infrastructure is ready, proceed with Workstreams D, F, and N simultaneously."
```

## IMPLEMENTATION ORDER (RECOMMENDED)

```
Phase 1 (Day 1-2):   A + B        → Core infra + DB schema
Phase 2 (Day 2-3):   C + D        → Auth + VCS integrations
Phase 3 (Day 3-5):   E + F + G    → Pipeline DSL + Engine + Agents
Phase 4 (Day 4-6):   H + I + K    → Detector + Secrets + Storage
Phase 5 (Day 5-7):   J + L + M    → Notifications + WebSocket + API handlers
Phase 6 (Day 6-8):   N + O        → Full frontend
Phase 7 (Day 7-9):   P + Q        → Deployment + CLI
Phase 8 (Day 9-10):  Testing + Docs
```

---

*FlowForge CLAUDE.md — Last updated: 2026. Keep this file updated as the codebase evolves.*
