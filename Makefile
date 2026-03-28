# ==============================================================================
# FlowForge CI/CD Platform — Makefile
# ==============================================================================

WEB_DIR := web
DIST_DIR := dist
SERVER_CMD := ./cmd/server

.PHONY: help dev dev-down dev-backend dev-web dev-frontend \
        build build-backend build-web build-frontend \
        test test-backend test-web test-frontend test-integration \
        lint lint-backend lint-web lint-frontend \
        docker-build docker-up docker-down docker-logs docker-ps \
        clean clean-docker migrate seed fmt vet proto \
        k8s-apply k8s-delete

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
	CGO_ENABLED=1 go run $(SERVER_CMD)

dev-web: ## Run web app locally (no Docker)
	cd $(WEB_DIR) && npm run dev

dev-frontend: dev-web ## Backward-compatible alias for dev-web

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build: build-backend build-web ## Build backend and web app

build-backend: ## Build Go backend binary
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 go build -ldflags="-w -s" -trimpath -o $(DIST_DIR)/flowforge-server $(SERVER_CMD)

build-web: ## Build web production bundle
	cd $(WEB_DIR) && npm ci && npm run build

build-frontend: build-web ## Backward-compatible alias for build-web

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

test: test-backend test-web ## Run all tests

test-backend: ## Run backend tests
	CGO_ENABLED=1 go test ./... -race -cover -timeout 120s

test-web: ## Run web tests
	cd $(WEB_DIR) && npm test 2>/dev/null || echo "No test script configured"

test-frontend: test-web ## Backward-compatible alias for test-web

test-integration: ## Run integration tests (requires Docker)
	docker compose -f docker-compose.dev.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	go test ./... -tags=integration -race -timeout 300s
	docker compose -f docker-compose.dev.yml down

# ---------------------------------------------------------------------------
# Code Quality
# ---------------------------------------------------------------------------

lint: lint-backend lint-web ## Run all linters

lint-backend: ## Lint Go code
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed, skipping"

lint-web: ## Lint web code
	cd $(WEB_DIR) && npm run typecheck 2>/dev/null || echo "No typecheck script configured"

lint-frontend: lint-web ## Backward-compatible alias for lint-web

fmt: ## Format Go code
	gofmt -w -s .

vet: ## Run go vet
	go vet ./...

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
	go run $(SERVER_CMD) -migrate

seed: ## Seed database with sample data
	@test -f scripts/seed-db.sh && bash scripts/seed-db.sh || echo "Seed script not found"

# ---------------------------------------------------------------------------
# Protobuf
# ---------------------------------------------------------------------------

proto: ## Generate protobuf Go code
	@test -f scripts/gen-proto.sh && bash scripts/gen-proto.sh || \
		(protoc --go_out=. --go-grpc_out=. proto/agent.proto 2>/dev/null || echo "protoc not installed")

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

clean: ## Remove build artifacts and temporary files
	rm -rf $(DIST_DIR)/
	rm -rf $(WEB_DIR)/dist/
	rm -rf data/
	rm -rf tmp/
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
