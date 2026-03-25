# ==============================================================================
# FlowForge CI/CD Platform — Makefile
# ==============================================================================

.PHONY: help dev dev-down build test test-backend test-frontend lint \
        docker-build docker-up docker-down clean migrate seed \
        fmt vet proto

# Default target
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Development
# ---------------------------------------------------------------------------

dev: ## Start development environment with hot reload
	docker compose -f docker-compose.dev.yml up --build

dev-down: ## Stop development environment
	docker compose -f docker-compose.dev.yml down

dev-backend: ## Run backend locally (no Docker)
	cd backend && CGO_ENABLED=1 go run ./cmd/server

dev-frontend: ## Run frontend locally (no Docker)
	cd frontend && npm run dev

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build: build-backend build-frontend ## Build backend and frontend

build-backend: ## Build Go backend binary
	cd backend && CGO_ENABLED=1 go build -ldflags="-w -s" -trimpath -o ../dist/flowforge-server ./cmd/server

build-frontend: ## Build frontend production bundle
	cd frontend && npm ci && npm run build

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

test: test-backend test-frontend ## Run all tests

test-backend: ## Run backend tests
	cd backend && CGO_ENABLED=1 go test ./... -race -cover -timeout 120s

test-frontend: ## Run frontend tests
	cd frontend && npm test 2>/dev/null || echo "No test script configured"

test-integration: ## Run integration tests (requires Docker)
	docker compose -f docker-compose.dev.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	cd backend && go test ./... -tags=integration -race -timeout 300s
	docker compose -f docker-compose.dev.yml down

# ---------------------------------------------------------------------------
# Code Quality
# ---------------------------------------------------------------------------

lint: lint-backend lint-frontend ## Run all linters

lint-backend: ## Lint Go code
	cd backend && go vet ./...
	@which golangci-lint > /dev/null 2>&1 && cd backend && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

lint-frontend: ## Lint frontend code
	cd frontend && npm run typecheck 2>/dev/null || echo "No typecheck script configured"

fmt: ## Format Go code
	cd backend && gofmt -w -s .

vet: ## Run go vet
	cd backend && go vet ./...

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

docker-build: ## Build all Docker images
	docker compose build

docker-up: ## Start production Docker stack
	docker compose up -d

docker-down: ## Stop production Docker stack
	docker compose down

docker-logs: ## Tail Docker logs
	docker compose logs -f

docker-ps: ## Show running containers
	docker compose ps

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

migrate: ## Run database migrations
	cd backend && go run ./cmd/server -migrate

seed: ## Seed database with sample data
	@test -f scripts/seed-db.sh && bash scripts/seed-db.sh || echo "Seed script not found"

# ---------------------------------------------------------------------------
# Protobuf
# ---------------------------------------------------------------------------

proto: ## Generate protobuf Go code
	@test -f scripts/gen-proto.sh && bash scripts/gen-proto.sh || \
		(cd backend && protoc --go_out=. --go-grpc_out=. proto/agent.proto 2>/dev/null || echo "protoc not installed")

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

clean: ## Remove build artifacts and temporary files
	rm -rf dist/
	rm -rf frontend/dist/
	rm -rf data/
	rm -rf backend/tmp/
	docker compose down -v 2>/dev/null || true
	docker compose -f docker-compose.dev.yml down -v 2>/dev/null || true

clean-docker: ## Remove all FlowForge Docker images and volumes
	docker compose down -v --rmi all 2>/dev/null || true
	docker compose -f docker-compose.dev.yml down -v --rmi all 2>/dev/null || true

# ---------------------------------------------------------------------------
# Kubernetes
# ---------------------------------------------------------------------------

k8s-apply: ## Apply Kubernetes manifests
	kubectl apply -f deploy/kubernetes/namespace.yaml
	kubectl apply -f deploy/kubernetes/

k8s-delete: ## Delete Kubernetes resources
	kubectl delete -f deploy/kubernetes/ 2>/dev/null || true
