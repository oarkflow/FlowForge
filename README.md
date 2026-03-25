# FlowForge

A self-hosted, horizontally-scalable CI/CD platform built with Go and SolidJS. Connect your repositories, define pipelines, and deploy with confidence.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Manual Setup](#manual-setup)
- [Configuration](#configuration)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Deployment](#deployment)
- [Development](#development)
- [Project Structure](#project-structure)

---

## Features

- **Multi-provider VCS integration** -- GitHub, GitLab, and Bitbucket via webhooks and OAuth
- **Flexible pipeline execution** -- Run locally, in Docker containers, or on Kubernetes
- **YAML/JSON pipeline DSL** -- Define stages, jobs, steps, matrix builds, conditional execution
- **Real-time log streaming** -- WebSocket with SSE fallback, ANSI color support
- **Built-in secret management** -- AES-256-GCM encrypted secrets with log scrubbing
- **Artifact storage** -- Local disk or S3-compatible backends (AWS S3, MinIO, GCS)
- **Notifications** -- Slack, Email, Microsoft Teams, Discord, PagerDuty, webhooks
- **RBAC** -- Owner, Admin, Developer, Viewer roles with project-level assignment
- **Language detection** -- Auto-detect Go, Node.js, Python, Ruby, Java, Rust, PHP, .NET, and more
- **Dark-themed dashboard** -- SolidJS real-time UI with pipeline visualization

## Architecture

```
                    ┌──────────────────────────────────────────┐
                    │            FlowForge Server               │
                    │                                          │
 ┌──────────┐      │  ┌─────────┐  ┌──────────┐  ┌────────┐  │
 │ SolidJS  │◄────►│  │  Fiber  │  │ Pipeline │  │  WS/   │  │
 │ Frontend │      │  │  REST   │  │  Engine  │  │  SSE   │  │
 └──────────┘      │  │  API    │  │          │  │  Hub   │  │
                    │  └────┬────┘  └────┬─────┘  └───┬────┘  │
 ┌──────────┐      │       │            │             │       │
 │ GitHub   │─────►│  ┌────▼────────────▼─────────────▼────┐  │
 │ GitLab   │      │  │         SQLite (WAL mode)          │  │
 │Bitbucket │      │  └───────────────────────────────────┘  │
 └──────────┘      └──────────────────────────────────────────┘
```

**Backend:** Go 1.22+ with GoFiber v3, jmoiron/sqlx, SQLite (WAL mode)
**Frontend:** SolidJS, TailwindCSS v4, TypeScript, Vite
**Database:** SQLite with Write-Ahead Logging (upgradeable to PostgreSQL)

---

## Requirements

### System Dependencies

| Dependency | Version | Required | Notes |
|---|---|---|---|
| **Go** | 1.22+ | Yes | CGO must be enabled for SQLite |
| **GCC** | Any | Yes | Required by go-sqlite3 (CGO) |
| **Node.js** | 20+ | Yes | For frontend build |
| **npm** | 9+ | Yes | Comes with Node.js |
| **Docker** | 24+ | Optional | For Docker executor and containerized dev |
| **Docker Compose** | v2+ | Optional | For full-stack development |
| **kubectl** | 1.28+ | Optional | For Kubernetes deployment |
| **protoc** | 3.x | Optional | For regenerating gRPC code |

### Operating Systems

- Linux (recommended for production)
- macOS (development)
- Windows with WSL2

---

## Quick Start

### Option 1: Docker Compose (Recommended)

The fastest way to get FlowForge running:

```bash
# Clone the repository
git clone https://github.com/oarkflow/deploy.git
cd deploy

# Copy environment config
cp .env.example .env

# Generate a secure encryption key
ENCRYPTION_KEY=$(openssl rand -hex 32)
sed -i.bak "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env

# Generate a secure JWT secret
JWT_SECRET=$(openssl rand -base64 32)
sed -i.bak "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env

# Start the full stack
make dev
```

This starts:
- **Backend** at `http://localhost:8081` (with hot reload via air)
- **Frontend** at `http://localhost:3000` (with Vite HMR)
- **MinIO** at `http://localhost:9001` (S3-compatible storage, login: minioadmin/minioadmin)

### Option 2: Run Locally (No Docker)

```bash
# Clone and enter the project
git clone https://github.com/oarkflow/deploy.git
cd deploy

# Copy and configure environment
cp .env.example .env
# Edit .env with your settings (see Configuration section)

# Terminal 1: Start the backend
cd backend
export CGO_ENABLED=1
export FLOWFORGE_PORT=8081
export FLOWFORGE_DATABASE_PATH=../data/flowforge.db
export FLOWFORGE_JWT_SECRET=$(openssl rand -base64 32)
export FLOWFORGE_ENCRYPTION_KEY=$(openssl rand -hex 32)
mkdir -p ../data
go run ./cmd/server

# Terminal 2: Start the frontend
cd frontend
npm install
VITE_API_URL=http://localhost:8081 npm run dev
```

Open `http://localhost:3000` in your browser.

---

## Manual Setup

### 1. Backend Setup

```bash
cd backend

# Verify Go and CGO
go version          # Must be 1.22+
gcc --version       # Required for go-sqlite3

# Install dependencies
go mod download

# Create data directory
mkdir -p ../data

# Run the server
CGO_ENABLED=1 go run ./cmd/server
```

The server automatically runs database migrations on startup. The SQLite database file is created at the configured `DATABASE_PATH`.

### 2. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Or build for production
npm run build
```

### 3. Verify Installation

```bash
# Check server health
curl http://localhost:8081/api/v1/system/health

# Expected response:
# {"status":"ok","timestamp":"..."}
```

---

## Configuration

FlowForge uses a layered configuration system with this precedence (highest to lowest):

1. **Environment variables** with `FLOWFORGE_` prefix
2. **config.yaml** file (searched in `.`, `./config`, `/etc/flowforge`)
3. **Built-in defaults**

### Environment Variables

#### Required

| Variable | Description | Default |
|---|---|---|
| `FLOWFORGE_JWT_SECRET` | JWT signing key (min 32 chars) | `change-me-in-production` |
| `FLOWFORGE_ENCRYPTION_KEY` | 64-char hex string for AES-256-GCM | Zeroed key (insecure) |

#### Server

| Variable | Description | Default |
|---|---|---|
| `FLOWFORGE_PORT` | HTTP server port | `8081` |
| `FLOWFORGE_LOG_LEVEL` | Log level: debug, info, warn, error | `info` |
| `FLOWFORGE_ALLOWED_ORIGINS` | CORS allowed origins | `*` |
| `FLOWFORGE_MAX_UPLOAD_SIZE` | Max upload size in bytes | `52428800` (50 MB) |

#### Database

| Variable | Description | Default |
|---|---|---|
| `FLOWFORGE_DATABASE_PATH` | SQLite database file path | `data/flowforge.db` |

The database runs in WAL (Write-Ahead Logging) mode with foreign keys enabled and a 5-second busy timeout. Migrations run automatically on startup.

#### Artifact Storage

| Variable | Description | Default |
|---|---|---|
| `FLOWFORGE_ARTIFACT_STORAGE_BACKEND` | `local` or `s3` | `local` |
| `FLOWFORGE_ARTIFACT_STORAGE_PATH` | Local storage directory | `/data/artifacts` |
| `FLOWFORGE_S3_ENDPOINT` | S3/MinIO endpoint URL | -- |
| `FLOWFORGE_S3_BUCKET` | S3 bucket name | -- |
| `FLOWFORGE_S3_ACCESS_KEY` | S3 access key | -- |
| `FLOWFORGE_S3_SECRET_KEY` | S3 secret key | -- |
| `FLOWFORGE_S3_USE_SSL` | Enable TLS for S3 | `false` |

#### VCS Integrations

<details>
<summary>GitHub</summary>

| Variable | Description |
|---|---|
| `FLOWFORGE_GITHUB_APP_ID` | GitHub App ID |
| `FLOWFORGE_GITHUB_APP_PRIVATE_KEY_PATH` | Path to GitHub App private key |
| `FLOWFORGE_GITHUB_CLIENT_ID` | OAuth client ID |
| `FLOWFORGE_GITHUB_CLIENT_SECRET` | OAuth client secret |
| `FLOWFORGE_GITHUB_WEBHOOK_SECRET` | Webhook HMAC-SHA256 secret |

</details>

<details>
<summary>GitLab</summary>

| Variable | Description |
|---|---|
| `FLOWFORGE_GITLAB_CLIENT_ID` | OAuth client ID |
| `FLOWFORGE_GITLAB_CLIENT_SECRET` | OAuth client secret |
| `FLOWFORGE_GITLAB_WEBHOOK_SECRET` | Webhook secret token |

</details>

<details>
<summary>Bitbucket</summary>

| Variable | Description |
|---|---|
| `FLOWFORGE_BITBUCKET_CLIENT_ID` | OAuth consumer key |
| `FLOWFORGE_BITBUCKET_CLIENT_SECRET` | OAuth consumer secret |
| `FLOWFORGE_BITBUCKET_WEBHOOK_SECRET` | Webhook secret |

</details>

#### Notifications

<details>
<summary>Email (SMTP)</summary>

| Variable | Description | Default |
|---|---|---|
| `FLOWFORGE_SMTP_HOST` | SMTP server hostname | -- |
| `FLOWFORGE_SMTP_PORT` | SMTP server port | `587` |
| `FLOWFORGE_SMTP_USER` | SMTP username | -- |
| `FLOWFORGE_SMTP_PASSWORD` | SMTP password | -- |
| `FLOWFORGE_SMTP_FROM` | Sender email address | `noreply@flowforge.local` |

</details>

<details>
<summary>Slack</summary>

Configure Slack notifications per project through the UI or API. Each channel requires a webhook URL stored encrypted in the database.

</details>

#### External Secret Providers

| Variable | Description |
|---|---|
| `FLOWFORGE_VAULT_ADDR` | HashiCorp Vault address |
| `FLOWFORGE_VAULT_TOKEN` | Vault authentication token |
| `FLOWFORGE_AWS_REGION` | AWS region for Secrets Manager |
| `FLOWFORGE_AWS_ACCESS_KEY_ID` | AWS access key |
| `FLOWFORGE_AWS_SECRET_ACCESS_KEY` | AWS secret key |

### Config File

You can also use a YAML config file:

```yaml
# config.yaml
port: "8081"
database_path: "data/flowforge.db"
jwt_secret: "your-secret-here"
encryption_key: "your-64-char-hex-key"
allowed_origins: "http://localhost:3000"
log_level: "info"
max_upload_size: 52428800
```

Place it in the project root, `./config/`, or `/etc/flowforge/`.

---

## Usage

### Authentication

#### Register a New User

```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "username": "admin",
    "password": "your-secure-password",
    "display_name": "Admin User"
  }'
```

#### Login

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your-secure-password"
  }'

# Response:
# {
#   "access_token": "eyJhbGci...",
#   "refresh_token": "eyJhbGci...",
#   "expires_in": 900
# }
```

Use the `access_token` as a Bearer token for subsequent requests:

```bash
export TOKEN="eyJhbGci..."
```

### Projects

```bash
# Create a project
curl -X POST http://localhost:8081/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My App",
    "slug": "my-app",
    "description": "My application project"
  }'

# List projects
curl http://localhost:8081/api/v1/projects \
  -H "Authorization: Bearer $TOKEN"
```

### Pipelines

```bash
# Create a pipeline with inline YAML config
curl -X POST http://localhost:8081/api/v1/projects/{project_id}/pipelines \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Build & Test",
    "config_source": "db",
    "config_content": "version: \"1\"\nname: Build & Test\nstages:\n  - test\njobs:\n  unit-tests:\n    stage: test\n    steps:\n      - name: Run tests\n        run: go test ./..."
  }'

# Trigger a pipeline run
curl -X POST http://localhost:8081/api/v1/projects/{project_id}/pipelines/{pipeline_id}/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "branch": "main",
    "trigger_type": "manual"
  }'
```

### Real-Time Log Streaming

#### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8081/ws/runs/{run_id}/logs');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // data.type: "log" | "replay_complete"
  // data.content: log line text
  // data.stream: "stdout" | "stderr" | "system"
  console.log(`[${data.stream}] ${data.content}`);
};
```

#### SSE Fallback

```javascript
const source = new EventSource('/sse/runs/{run_id}/logs');

source.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.content);
};
```

### Secrets

```bash
# Create a secret
curl -X POST http://localhost:8081/api/v1/projects/{project_id}/secrets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "DEPLOY_TOKEN",
    "value": "ghp_xxxxxxxxxxxx"
  }'

# List secrets (values are never returned)
curl http://localhost:8081/api/v1/projects/{project_id}/secrets \
  -H "Authorization: Bearer $TOKEN"
```

### Agents

```bash
# Register an agent
curl -X POST http://localhost:8081/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-linux-01",
    "executor": "docker",
    "labels": ["linux", "docker", "large"]
  }'

# List agents with status
curl http://localhost:8081/api/v1/agents \
  -H "Authorization: Bearer $TOKEN"

# Drain an agent (stop accepting new jobs)
curl -X POST http://localhost:8081/api/v1/agents/{agent_id}/drain \
  -H "Authorization: Bearer $TOKEN"
```

### Notifications

```bash
# Add a Slack notification channel to a project
curl -X POST http://localhost:8081/api/v1/projects/{project_id}/notifications \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "slack",
    "name": "Build Alerts",
    "config": {
      "webhook_url": "https://hooks.slack.com/services/T00/B00/xxx",
      "channel": "#builds"
    }
  }'
```

---

## API Reference

### Authentication
| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login, returns JWT tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Invalidate session |

### Users
| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/users/me` | JWT | Get current user profile |
| PUT | `/api/v1/users/me` | JWT | Update current user |
| GET | `/api/v1/users/:id` | JWT | Get user by ID |

### Organizations
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/orgs` | Any | List organizations |
| POST | `/api/v1/orgs` | Admin | Create organization |
| GET | `/api/v1/orgs/:id` | Any | Get organization |
| PUT | `/api/v1/orgs/:id` | Admin | Update organization |
| DELETE | `/api/v1/orgs/:id` | Owner | Delete organization |
| GET | `/api/v1/orgs/:id/members` | Any | List members |
| POST | `/api/v1/orgs/:id/members` | Admin | Add member |
| DELETE | `/api/v1/orgs/:id/members/:userId` | Admin | Remove member |

### Projects
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/projects` | Any | List projects |
| POST | `/api/v1/projects` | Developer+ | Create project |
| GET | `/api/v1/projects/:id` | Any | Get project |
| PUT | `/api/v1/projects/:id` | Developer+ | Update project |
| DELETE | `/api/v1/projects/:id` | Admin | Delete project |

### Repositories
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/projects/:id/repositories` | Any | List repositories |
| POST | `/api/v1/projects/:id/repositories` | Developer+ | Connect repository |
| DELETE | `/api/v1/projects/:id/repositories/:repoId` | Admin | Disconnect repository |
| POST | `/api/v1/projects/:id/repositories/:repoId/sync` | Developer+ | Sync repository |

### Pipelines
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/projects/:id/pipelines` | Any | List pipelines |
| POST | `/api/v1/projects/:id/pipelines` | Developer+ | Create pipeline |
| GET | `/api/v1/projects/:id/pipelines/:pid` | Any | Get pipeline |
| PUT | `/api/v1/projects/:id/pipelines/:pid` | Developer+ | Update pipeline |
| DELETE | `/api/v1/projects/:id/pipelines/:pid` | Admin | Delete pipeline |
| GET | `/api/v1/projects/:id/pipelines/:pid/versions` | Any | List versions |
| POST | `/api/v1/projects/:id/pipelines/:pid/trigger` | Developer+ | Trigger run |
| POST | `/api/v1/projects/:id/pipelines/:pid/validate` | Developer+ | Validate config |

### Pipeline Runs
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/projects/:id/pipelines/:pid/runs` | Any | List runs |
| GET | `/api/v1/projects/:id/pipelines/:pid/runs/:rid` | Any | Get run detail |
| POST | `/api/v1/projects/:id/pipelines/:pid/runs/:rid/cancel` | Developer+ | Cancel run |
| POST | `/api/v1/projects/:id/pipelines/:pid/runs/:rid/rerun` | Developer+ | Re-run pipeline |
| POST | `/api/v1/projects/:id/pipelines/:pid/runs/:rid/approve` | Admin | Approve gated run |
| GET | `/api/v1/projects/:id/pipelines/:pid/runs/:rid/logs` | Any | Get run logs |
| GET | `/api/v1/projects/:id/pipelines/:pid/runs/:rid/artifacts` | Any | List artifacts |

### Secrets
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/projects/:id/secrets` | Developer+ | List secrets (no values) |
| POST | `/api/v1/projects/:id/secrets` | Admin | Create secret |
| PUT | `/api/v1/projects/:id/secrets/:secretId` | Admin | Update secret |
| DELETE | `/api/v1/projects/:id/secrets/:secretId` | Admin | Delete secret |

### Agents
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/agents` | Any | List agents |
| POST | `/api/v1/agents` | Admin | Register agent |
| GET | `/api/v1/agents/:id` | Any | Get agent |
| DELETE | `/api/v1/agents/:id` | Admin | Remove agent |
| POST | `/api/v1/agents/:id/drain` | Admin | Drain agent |

### Webhooks (Unauthenticated, Signature-Validated)
| Method | Endpoint | Description |
|---|---|---|
| POST | `/webhooks/github` | GitHub webhook receiver |
| POST | `/webhooks/gitlab` | GitLab webhook receiver |
| POST | `/webhooks/bitbucket` | Bitbucket webhook receiver |

### WebSocket / SSE
| Protocol | Endpoint | Description |
|---|---|---|
| WS | `/ws/runs/:runId/logs` | Real-time log streaming for a run |
| WS | `/ws/events` | System-wide event stream |
| SSE | `/sse/runs/:runId/logs` | SSE fallback for log streaming |

### System
| Method | Endpoint | Role | Description |
|---|---|---|---|
| GET | `/api/v1/system/health` | Public | Health check |
| GET | `/api/v1/system/metrics` | Admin | System metrics |
| GET | `/api/v1/system/info` | Admin | System info |

---

## Deployment

### Production with Docker Compose

```bash
# Configure environment
cp .env.example .env
# Edit .env with production values:
#   - Strong JWT_SECRET (openssl rand -base64 32)
#   - Random ENCRYPTION_KEY (openssl rand -hex 32)
#   - Restricted ALLOWED_ORIGINS

# Build and start
make docker-build
make docker-up

# View logs
make docker-logs

# Stop
make docker-down
```

### Kubernetes

```bash
# 1. Create namespace and apply manifests
make k8s-apply

# 2. Create the secrets (edit values first)
kubectl -n flowforge create secret generic flowforge-secrets \
  --from-literal=jwt-secret="$(openssl rand -base64 32)" \
  --from-literal=encryption-key="$(openssl rand -hex 32)"

# 3. Verify deployment
kubectl -n flowforge get pods
kubectl -n flowforge logs -f deployment/flowforge-server

# Cleanup
make k8s-delete
```

Kubernetes manifests are in `deploy/kubernetes/`:
- `namespace.yaml` -- FlowForge namespace
- `deployment-server.yaml` -- 3-replica backend with health checks
- `service.yaml` -- ClusterIP service
- `ingress.yaml` -- Ingress with TLS

### Build Production Binaries

```bash
# Build everything
make build

# Output:
#   dist/flowforge-server     (Go binary, CGO-enabled)
#   frontend/dist/            (Static SPA bundle)

# Run the server binary directly
./dist/flowforge-server
```

---

## Development

### Make Targets

```bash
make help              # Show all available targets

# Development
make dev               # Start full dev stack (Docker Compose)
make dev-down          # Stop dev stack
make dev-backend       # Run backend only (no Docker)
make dev-frontend      # Run frontend only (no Docker)

# Build
make build             # Build backend + frontend
make build-backend     # Build Go binary to dist/flowforge-server
make build-frontend    # Build frontend production bundle

# Testing
make test              # Run all tests
make test-backend      # Go tests with race detection and coverage
make test-frontend     # Frontend tests
make test-integration  # Integration tests (requires Docker)

# Code Quality
make lint              # Run all linters
make fmt               # Format Go code
make vet               # Run go vet

# Docker
make docker-build      # Build all Docker images
make docker-up         # Start production stack
make docker-down       # Stop production stack
make docker-logs       # Tail container logs

# Database
make migrate           # Run database migrations
make seed              # Seed with sample data

# Cleanup
make clean             # Remove all build artifacts and data
make clean-docker      # Remove all Docker images and volumes
```

### Pipeline YAML Syntax

FlowForge uses a YAML-based pipeline definition. Here's a minimal example:

```yaml
version: "1"
name: "Build & Test"

on:
  push:
    branches: ["main", "develop"]
  pull_request:
    types: [opened, synchronize]

stages:
  - test
  - build

jobs:
  unit-tests:
    stage: test
    steps:
      - name: Run tests
        run: go test ./... -race -cover

  build-binary:
    stage: build
    needs: [unit-tests]
    steps:
      - name: Build
        run: CGO_ENABLED=0 go build -o dist/app ./cmd/server
```

#### Advanced Features

```yaml
# Matrix builds
jobs:
  test-matrix:
    stage: test
    matrix:
      go_version: ["1.21", "1.22"]
      os: [ubuntu, alpine]
    image: golang:${{ matrix.go_version }}
    steps:
      - run: go test ./...

# Conditional execution
jobs:
  deploy:
    stage: deploy
    when: ${{ git.branch == 'main' }}
    steps:
      - name: Deploy to production
        run: ./deploy.sh

# Retry and timeout
jobs:
  flaky-test:
    stage: test
    timeout: 10m
    retry:
      count: 3
      delay: 30s
    steps:
      - run: npm run test:e2e

# Environment variables and secrets
jobs:
  build:
    stage: build
    env:
      NODE_ENV: production
    steps:
      - name: Build with secrets
        run: npm run build
        env:
          API_KEY: ${{ secrets.API_KEY }}
```

### Database

FlowForge uses SQLite in WAL mode. The database is a single file that can be backed up by copying it.

**Schema includes 12 tables:**

| Table | Purpose |
|---|---|
| `users` | User accounts with RBAC roles |
| `organizations` | Team/org management |
| `org_members` | Organization membership |
| `projects` | Project containers |
| `repositories` | VCS connections |
| `pipelines` | Pipeline definitions |
| `pipeline_versions` | Version history |
| `pipeline_runs` | Execution records |
| `stage_runs` / `job_runs` / `step_runs` | Granular execution tracking |
| `run_logs` | Build log storage |
| `agents` | Worker agent registry |
| `secrets` | Encrypted secret store |
| `artifacts` | Build artifact metadata |
| `notification_channels` | Notification configs |
| `audit_logs` | Mutation audit trail |

Migrations are embedded in the binary and run automatically on startup.

### RBAC Roles

| Role | Level | Permissions |
|---|---|---|
| **Viewer** | 1 | Read-only access to projects, pipelines, runs, logs |
| **Developer** | 2 | Create/edit pipelines, trigger runs, manage secrets |
| **Admin** | 3 | Manage users, agents, notifications, delete resources |
| **Owner** | 4 | Full access including organization management |

---

## Project Structure

```
deploy/
├── README.md                    # This file
├── CLAUDE.md                    # Full project specification
├── Makefile                     # Build and development targets
├── docker-compose.yml           # Production Docker stack
├── docker-compose.dev.yml       # Development stack with hot reload
├── .env.example                 # Environment variable template
│
├── backend/                     # Go backend
│   ├── cmd/server/main.go       # Application entry point
│   ├── internal/
│   │   ├── api/                 # HTTP handlers, middleware, routing
│   │   ├── auth/                # JWT, OAuth, TOTP, API keys
│   │   ├── config/              # Configuration loading (Viper)
│   │   ├── db/                  # SQLx database, migrations, queries
│   │   ├── detector/            # Language/framework auto-detection
│   │   ├── engine/              # Pipeline execution engine
│   │   ├── integrations/        # GitHub, GitLab, Bitbucket clients
│   │   ├── models/              # Domain model structs
│   │   ├── notifications/       # Event bus, Slack, Email, Teams, etc.
│   │   ├── pipeline/            # YAML/JSON DSL parser and validator
│   │   ├── secrets/             # Encrypted secret store, log scrubbing
│   │   ├── storage/             # Artifact backends (local, S3)
│   │   └── websocket/           # Real-time log streaming hub
│   └── pkg/                     # Reusable packages (crypto, logger)
│
├── frontend/                    # SolidJS frontend
│   └── src/
│       ├── api/                 # Typed API client
│       ├── components/          # UI components and layouts
│       ├── pages/               # Route pages
│       ├── stores/              # SolidJS reactive stores
│       └── types/               # TypeScript type definitions
│
└── deploy/                      # Infrastructure
    ├── docker/                  # Dockerfiles (backend, frontend, agent)
    └── kubernetes/              # K8s manifests
```

---

## Security

- **Secrets** are encrypted at rest using AES-256-GCM. Secret values are never returned by the API and are automatically scrubbed from build logs.
- **Webhook signatures** are validated (HMAC-SHA256 for GitHub, token matching for GitLab/Bitbucket) before processing events.
- **JWT tokens** expire after 15 minutes. Refresh tokens are valid for 7 days.
- **CORS** is configurable via `ALLOWED_ORIGINS`.
- **Rate limiting** is applied globally (100 requests/minute by default).
- **Security headers** are set via Helmet middleware (X-Frame-Options, Content-Security-Policy, etc.).
- **Audit logging** records all mutations with actor, timestamp, and change diff.

### Generating Secure Keys

```bash
# Generate encryption key (64-char hex = 32 bytes)
openssl rand -hex 32

# Generate JWT secret
openssl rand -base64 32
```

---

## License

See [LICENSE](LICENSE) for details.
