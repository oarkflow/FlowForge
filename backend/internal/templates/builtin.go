package templates

import "github.com/oarkflow/deploy/backend/internal/models"

// BuiltinTemplates contains the set of built-in pipeline templates that ship with FlowForge.
var BuiltinTemplates = []models.PipelineTemplate{
	{
		ID:          "builtin-go-ci",
		Name:        "Go CI",
		Description: "Standard CI pipeline for Go projects: lint, test, and build.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Go CI"
on:
  push:
    branches: ["main", "develop"]
  pull_request:
    types: [opened, synchronize]
stages:
  - lint
  - test
  - build
jobs:
  lint:
    stage: lint
    executor: docker
    image: golangci/golangci-lint:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Lint
        run: golangci-lint run ./...
  test:
    stage: test
    executor: docker
    image: golang:1.22-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: go test -race -coverprofile=coverage.out ./...
      - uses: flowforge/upload-artifact@v1
        with:
          name: coverage
          path: coverage.out
  build:
    stage: build
    executor: docker
    image: golang:1.22-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Build
        run: CGO_ENABLED=0 go build -ldflags="-w -s" -o app ./cmd/server
`,
	},
	{
		ID:          "builtin-node-ci",
		Name:        "Node.js CI",
		Description: "CI pipeline for Node.js projects: install, lint, test, and build.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Node.js CI"
on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]
stages:
  - install
  - lint
  - test
  - build
jobs:
  install:
    stage: install
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Install dependencies
        run: npm ci
  lint:
    stage: lint
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Lint
        run: npm run lint
  test:
    stage: test
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: npm test
  build:
    stage: build
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Build
        run: npm run build
`,
	},
	{
		ID:          "builtin-python-ci",
		Name:        "Python CI",
		Description: "CI pipeline for Python projects: lint, test, and type check.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Python CI"
on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]
stages:
  - setup
  - lint
  - test
jobs:
  setup:
    stage: setup
    executor: docker
    image: python:3.12-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Install dependencies
        run: pip install -r requirements.txt
  lint:
    stage: lint
    executor: docker
    image: python:3.12-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Lint
        run: |
          pip install ruff
          ruff check .
  test:
    stage: test
    executor: docker
    image: python:3.12-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: |
          pip install -r requirements.txt pytest
          pytest --tb=short -v
`,
	},
	{
		ID:          "builtin-docker-build",
		Name:        "Docker Build & Push",
		Description: "Build a Docker image and push to a container registry.",
		Category:    "build",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Docker Build & Push"
on:
  push:
    branches: ["main"]
  manual: {}
stages:
  - build
jobs:
  docker:
    stage: build
    executor: docker
    image: docker:26-dind
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Build Docker image
        run: docker build -t $IMAGE_NAME:$IMAGE_TAG .
      - name: Push Docker image
        run: |
          echo "$REGISTRY_PASSWORD" | docker login $REGISTRY_URL -u "$REGISTRY_USERNAME" --password-stdin
          docker push $IMAGE_NAME:$IMAGE_TAG
`,
	},
	{
		ID:          "builtin-k8s-deploy",
		Name:        "Kubernetes Deploy",
		Description: "Deploy to Kubernetes using Helm.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Kubernetes Deploy"
on:
  manual:
    inputs:
      environment:
        description: "Target environment"
        required: true
        type: choice
        options: [staging, production]
stages:
  - deploy
jobs:
  helm-deploy:
    stage: deploy
    executor: docker
    image: alpine/helm:3.14
    steps:
      - uses: flowforge/checkout@v1
      - name: Deploy with Helm
        uses: flowforge/helm-deploy@v1
        with:
          chart: ./deploy/helm/app
          release: app
          namespace: $NAMESPACE
`,
	},
	{
		ID:          "builtin-security-scan",
		Name:        "Security Scan",
		Description: "Run Trivy container and filesystem security scanning.",
		Category:    "security",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Security Scan"
on:
  push:
    branches: ["main"]
  schedule:
    - cron: "0 6 * * 1"
stages:
  - scan
jobs:
  trivy:
    stage: scan
    executor: docker
    image: aquasec/trivy:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Filesystem scan
        run: trivy fs --severity HIGH,CRITICAL --exit-code 1 .
`,
	},
	{
		ID:          "builtin-rust-ci",
		Name:        "Rust CI",
		Description: "CI pipeline for Rust projects: check, clippy, test, and build.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Rust CI"
on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]
stages:
  - check
  - test
  - build
jobs:
  check:
    stage: check
    executor: docker
    image: rust:1.77-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Check
        run: cargo check
      - name: Clippy
        run: cargo clippy -- -D warnings
  test:
    stage: test
    executor: docker
    image: rust:1.77-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: cargo test
  build:
    stage: build
    executor: docker
    image: rust:1.77-slim
    steps:
      - uses: flowforge/checkout@v1
      - name: Build release
        run: cargo build --release
`,
	},
	{
		ID:          "builtin-java-maven-ci",
		Name:        "Java Maven CI",
		Description: "CI pipeline for Java projects using Maven.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Java Maven CI"
on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]
stages:
  - test
  - build
jobs:
  test:
    stage: test
    executor: docker
    image: maven:3.9-eclipse-temurin-21
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: mvn test -B
  build:
    stage: build
    executor: docker
    image: maven:3.9-eclipse-temurin-21
    steps:
      - uses: flowforge/checkout@v1
      - name: Build
        run: mvn package -DskipTests -B
`,
	},
	{
		ID:          "builtin-terraform",
		Name:        "Terraform Plan & Apply",
		Description: "Run Terraform plan and apply for infrastructure as code.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Terraform Plan & Apply"
on:
  push:
    branches: ["main"]
    paths: ["terraform/**"]
  manual: {}
stages:
  - plan
  - apply
jobs:
  plan:
    stage: plan
    executor: docker
    image: hashicorp/terraform:1.7
    steps:
      - uses: flowforge/checkout@v1
      - name: Init
        run: terraform init
      - name: Plan
        run: terraform plan -out=tfplan
  apply:
    stage: apply
    executor: docker
    image: hashicorp/terraform:1.7
    approval_required: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Apply
        run: terraform apply tfplan
`,
	},
	{
		ID:          "builtin-php-ci",
		Name:        "PHP CI",
		Description: "CI pipeline for PHP projects with Composer and PHPUnit.",
		Category:    "ci",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "PHP CI"
on:
  push:
    branches: ["main"]
  pull_request:
    types: [opened, synchronize]
stages:
  - install
  - test
jobs:
  install:
    stage: install
    executor: docker
    image: composer:2
    steps:
      - uses: flowforge/checkout@v1
      - name: Install
        run: composer install --no-interaction --prefer-dist
  test:
    stage: test
    executor: docker
    image: php:8.3-cli
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: vendor/bin/phpunit
`,
	},
	{
		ID:          "builtin-static-site",
		Name:        "Static Site Deploy",
		Description: "Build and deploy a static site (React, Vue, Svelte, etc.).",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Static Site Deploy"
on:
  push:
    branches: ["main"]
stages:
  - build
  - deploy
jobs:
  build:
    stage: build
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Install
        run: npm ci
      - name: Build
        run: npm run build
      - uses: flowforge/upload-artifact@v1
        with:
          name: dist
          path: dist/
  deploy:
    stage: deploy
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/download-artifact@v1
        with:
          name: dist
          path: dist/
      - name: Deploy
        run: echo "Deploy dist/ to your hosting provider"
`,
	},
	// =====================================================================
	// Sample project templates — real-world CI/CD pipeline examples
	// =====================================================================
	{
		ID:          "builtin-laravel-vps-deploy",
		Name:        "Laravel PHP — VPS Deploy",
		Description: "Full CI/CD for Laravel: composer install, test, zip, deploy to VPS Nginx, migrate database, update .env, and cleanup.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Laravel PHP — VPS Deploy"
on:
  push:
    branches: ["main", "develop"]
  pull_request:
    types: [opened, synchronize]
stages:
  - install
  - test
  - package
  - deploy
  - post-deploy
  - cleanup
jobs:
  install:
    stage: install
    executor: docker
    image: composer:2
    steps:
      - uses: flowforge/checkout@v1
      - name: Install PHP dependencies
        run: composer install --no-interaction --prefer-dist --optimize-autoloader --no-dev
  test:
    stage: test
    executor: docker
    image: php:8.3-cli
    steps:
      - uses: flowforge/checkout@v1
      - name: Install dev dependencies
        run: |
          apt-get update && apt-get install -y unzip git libzip-dev
          docker-php-ext-install zip pdo pdo_mysql
          curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer
          composer install --no-interaction --prefer-dist
      - name: Run tests
        run: |
          cp .env.example .env
          php artisan key:generate
          php artisan test --parallel
  package:
    stage: package
    executor: docker
    image: php:8.3-cli
    steps:
      - uses: flowforge/checkout@v1
      - name: Create deployment archive
        run: |
          apt-get update && apt-get install -y zip
          zip -r "release-$(date +%Y%m%d%H%M%S).zip" . -x "*.git*" -x "tests/*" -x "node_modules/*"
      - name: Upload release
        uses: flowforge/upload-artifact@v1
        with:
          name: laravel-release
          path: release-*.zip
  deploy:
    stage: deploy
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - uses: flowforge/download-artifact@v1
        with:
          name: laravel-release
          path: ./
      - name: Deploy to VPS
        run: |
          apk add --no-cache openssh-client
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh-keyscan -H "$VPS_HOST" >> ~/.ssh/known_hosts 2>/dev/null
          scp release-*.zip "${VPS_USER}@${VPS_HOST}:/tmp/"
          ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p /var/www/html/releases/$(date +%Y%m%d%H%M%S) && sudo unzip -o /tmp/release-*.zip -d /var/www/html/releases/$(date +%Y%m%d%H%M%S) && sudo ln -nfs /var/www/html/releases/$(date +%Y%m%d%H%M%S) /var/www/html/current"
  migrate:
    stage: post-deploy
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Run migrations and update .env
        run: |
          apk add --no-cache openssh-client
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh "${VPS_USER}@${VPS_HOST}" "cd /var/www/html/current && sudo -u www-data php artisan migrate --force && sudo -u www-data php artisan config:cache && sudo -u www-data php artisan route:cache"
  cleanup:
    stage: cleanup
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Clean old releases
        run: |
          apk add --no-cache openssh-client
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh "${VPS_USER}@${VPS_HOST}" "cd /var/www/html/releases && ls -1d */ | head -n -5 | xargs -r sudo rm -rf && sudo nginx -t && sudo systemctl reload nginx"
`,
	},
	{
		ID:          "builtin-golang-vps-deploy",
		Name:        "Go HTTP Server — VPS Deploy",
		Description: "Full CI/CD for a Go HTTP server with frontend: build SPA, embed assets, build binary, deploy to VPS with Nginx, migrate database, update .env.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Go HTTP Server — VPS Deploy"
on:
  push:
    branches: ["main", "develop"]
  pull_request:
    types: [opened, synchronize]
stages:
  - build-frontend
  - prepare
  - test
  - build
  - deploy
  - post-deploy
  - cleanup
jobs:
  build-frontend:
    stage: build-frontend
    executor: docker
    image: node:20-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Build frontend
        run: cd frontend && npm ci && npm run build
      - name: Upload frontend dist
        uses: flowforge/upload-artifact@v1
        with:
          name: frontend-dist
          path: frontend/dist/
  prepare:
    stage: prepare
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Download frontend
        uses: flowforge/download-artifact@v1
        with:
          name: frontend-dist
          path: frontend/dist/
      - name: Move assets to Go project root
        run: |
          mkdir -p static/ && cp -r frontend/dist/* static/
          mkdir -p dist-migrations/ && cp -r migrations/* dist-migrations/
          mkdir -p dist-templates/email/ && cp -r templates/email/* dist-templates/email/
      - name: Upload prepared assets
        uses: flowforge/upload-artifact@v1
        with:
          name: go-prepared-assets
          path: |
            static/
            dist-migrations/
            dist-templates/
  test:
    stage: test
    executor: docker
    image: golang:1.22-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Test
        run: go mod download && go test -race -coverprofile=coverage.out ./...
  build:
    stage: build
    executor: docker
    image: golang:1.22-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Download prepared assets
        uses: flowforge/download-artifact@v1
        with:
          name: go-prepared-assets
          path: ./
      - name: Build binary
        run: CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/server ./cmd/server
      - name: Package release
        run: |
          apk add --no-cache tar gzip
          tar -czf "release-$(date +%Y%m%d%H%M%S).tar.gz" dist/server static/ dist-migrations/ dist-templates/
      - name: Upload release
        uses: flowforge/upload-artifact@v1
        with:
          name: go-release
          path: release-*.tar.gz
  deploy:
    stage: deploy
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Download release
        uses: flowforge/download-artifact@v1
        with:
          name: go-release
          path: ./
      - name: Deploy to VPS
        run: |
          apk add --no-cache openssh-client tar gzip
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh-keyscan -H "$VPS_HOST" >> ~/.ssh/known_hosts 2>/dev/null
          scp release-*.tar.gz "${VPS_USER}@${VPS_HOST}:/tmp/"
          ssh "${VPS_USER}@${VPS_HOST}" "sudo mkdir -p /opt/goapp/releases/$(date +%Y%m%d%H%M%S) && sudo tar -xzf /tmp/release-*.tar.gz -C /opt/goapp/releases/$(date +%Y%m%d%H%M%S) && sudo ln -nfs /opt/goapp/releases/$(date +%Y%m%d%H%M%S) /opt/goapp/current && sudo systemctl restart goapp"
  migrate:
    stage: post-deploy
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Run migrations and update .env
        run: |
          apk add --no-cache openssh-client
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh "${VPS_USER}@${VPS_HOST}" "cd /opt/goapp/current && ./server migrate up"
  cleanup:
    stage: cleanup
    executor: docker
    image: alpine:latest
    steps:
      - uses: flowforge/checkout@v1
      - name: Clean old releases
        run: |
          apk add --no-cache openssh-client
          mkdir -p ~/.ssh && echo "$VPS_SSH_KEY" > ~/.ssh/id_rsa && chmod 600 ~/.ssh/id_rsa
          ssh "${VPS_USER}@${VPS_HOST}" "cd /opt/goapp/releases && ls -1d */ | head -n -5 | xargs -r sudo rm -rf && sudo nginx -t && sudo systemctl reload nginx"
`,
	},
	{
		ID:          "builtin-docker-multi-service",
		Name:        "Docker Multi-Service — Laravel + Go",
		Description: "Deploy Laravel PHP and a Go HTTP server in Docker with MySQL, PostgreSQL, and Redis databases using docker-compose.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Docker Multi-Service — Laravel + Go"
on:
  push:
    branches: ["main"]
  manual: {}
stages:
  - test
  - build-images
  - deploy
  - post-deploy
  - verify
jobs:
  test-laravel:
    stage: test
    executor: docker
    image: php:8.3-cli
    steps:
      - uses: flowforge/checkout@v1
      - name: Test Laravel
        run: |
          apt-get update && apt-get install -y unzip git libzip-dev
          docker-php-ext-install zip pdo pdo_mysql
          curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer
          cd laravel && composer install --no-interaction && cp .env.example .env && php artisan key:generate && php artisan test
  test-golang:
    stage: test
    executor: docker
    image: golang:1.22-alpine
    steps:
      - uses: flowforge/checkout@v1
      - name: Test Go
        run: cd golang && go mod download && go test -race ./...
  build-images:
    stage: build-images
    executor: docker
    image: docker:26-cli
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Build Docker images
        run: |
          docker build -t myapp-laravel:latest -f laravel/Dockerfile laravel/
          docker build -t myapp-golang:latest -f golang/Dockerfile golang/
  deploy:
    stage: deploy
    executor: docker
    image: docker:26-cli
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Deploy stack
        run: |
          docker-compose -f docker-compose.yml up -d --remove-orphans
          sleep 15
          docker-compose ps
  migrate:
    stage: post-deploy
    executor: docker
    image: docker:26-cli
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Run migrations
        run: |
          docker-compose exec -T laravel php artisan migrate --force
          docker-compose exec -T laravel php artisan config:cache
          docker-compose exec -T golang ./server migrate up
  verify:
    stage: verify
    executor: docker
    image: docker:26-cli
    privileged: true
    steps:
      - uses: flowforge/checkout@v1
      - name: Health check
        run: |
          docker-compose ps
          docker-compose exec -T laravel php artisan --version
          docker-compose exec -T golang wget -qO- http://localhost:8080/health
`,
	},
	{
		ID:          "builtin-laravel-scala-eb",
		Name:        "Laravel + Scala — Elastic Beanstalk",
		Description: "Multi-tier deploy: package Scala API Interface JAR, build Scala WAR Service (depends on API JAR), composer install + test + zip Laravel Web, deploy both to AWS Elastic Beanstalk (PHP + Tomcat), run migrations.",
		Category:    "deploy",
		IsBuiltin:   1,
		IsPublic:    1,
		Author:      "FlowForge",
		Config: `version: "1"
name: "Laravel + Scala — Elastic Beanstalk"
on:
  push:
    branches: ["main"]
  manual: {}
stages:
  - build-api-interface
  - build-scala-service
  - build-web
  - deploy-web
  - deploy-service
  - post-deploy
  - verify
jobs:
  build-api-interface:
    stage: build-api-interface
    executor: docker
    image: sbtscala/scala-sbt:eclipse-temurin-17.0.15_6_1.12.7_2.13.18
    steps:
      - uses: flowforge/checkout@v1
      - name: Package API Interface JAR
        run: cd scala-api-interface && sbt test package
      - name: Upload API JAR
        uses: flowforge/upload-artifact@v1
        with:
          name: api-interface-jar
          path: scala-api-interface/target/scala-2.13/*.jar
  build-scala-service:
    stage: build-scala-service
    executor: docker
    image: sbtscala/scala-sbt:eclipse-temurin-17.0.15_6_1.12.7_2.13.18
    steps:
      - uses: flowforge/checkout@v1
      - name: Download API JAR
        uses: flowforge/download-artifact@v1
        with:
          name: api-interface-jar
          path: /tmp/
      - name: Move API JAR to service lib/
        run: mkdir -p scala-service/lib && cp /tmp/*.jar scala-service/lib/
      - name: Build and package WAR
        run: cd scala-service && sbt test package
      - name: Upload WAR
        uses: flowforge/upload-artifact@v1
        with:
          name: scala-service-war
          path: scala-service/target/scala-2.13/*.war
  build-web:
    stage: build-web
    executor: docker
    image: php:8.3-cli
    steps:
      - uses: flowforge/checkout@v1
      - name: Install, test, and zip Laravel
        run: |
          apt-get update && apt-get install -y unzip git libzip-dev zip
          docker-php-ext-install zip pdo pdo_mysql
          curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer
          cd web && composer install --no-interaction
          cp .env.example .env && php artisan key:generate && php artisan test
          composer install --no-interaction --optimize-autoloader --no-dev
          zip -r "/tmp/web-$(date +%Y%m%d%H%M%S).zip" . -x "*.git*" -x "tests/*"
      - name: Upload Web ZIP
        uses: flowforge/upload-artifact@v1
        with:
          name: web-zip
          path: /tmp/web-*.zip
  deploy-web:
    stage: deploy-web
    executor: docker
    image: amazon/aws-cli:2.15.0
    steps:
      - uses: flowforge/checkout@v1
      - name: Download web ZIP
        uses: flowforge/download-artifact@v1
        with:
          name: web-zip
          path: /tmp/
      - name: Deploy Laravel to Elastic Beanstalk
        run: |
          VERSION="web-$(date +%Y%m%d%H%M%S)"
          aws s3 cp /tmp/web-*.zip "s3://${S3_BUCKET}/releases/web/${VERSION}.zip"
          aws elasticbeanstalk create-application-version --application-name "$EB_WEB_APP" --version-label "$VERSION" --source-bundle "S3Bucket=${S3_BUCKET},S3Key=releases/web/${VERSION}.zip" --region "$AWS_REGION"
          aws elasticbeanstalk update-environment --application-name "$EB_WEB_APP" --environment-name "$EB_WEB_ENV" --version-label "$VERSION" --region "$AWS_REGION"
          aws elasticbeanstalk wait environment-updated --application-name "$EB_WEB_APP" --environment-names "$EB_WEB_ENV" --region "$AWS_REGION"
  deploy-service:
    stage: deploy-service
    executor: docker
    image: amazon/aws-cli:2.15.0
    steps:
      - uses: flowforge/checkout@v1
      - name: Download Scala WAR
        uses: flowforge/download-artifact@v1
        with:
          name: scala-service-war
          path: /tmp/
      - name: Deploy Scala WAR to Elastic Beanstalk (Tomcat)
        run: |
          VERSION="service-$(date +%Y%m%d%H%M%S)"
          mkdir -p /tmp/eb-bundle && cp /tmp/*.war /tmp/eb-bundle/ROOT.war
          cd /tmp/eb-bundle && zip -r "/tmp/${VERSION}.zip" .
          aws s3 cp "/tmp/${VERSION}.zip" "s3://${S3_BUCKET}/releases/service/${VERSION}.zip"
          aws elasticbeanstalk create-application-version --application-name "$EB_SERVICE_APP" --version-label "$VERSION" --source-bundle "S3Bucket=${S3_BUCKET},S3Key=releases/service/${VERSION}.zip" --region "$AWS_REGION"
          aws elasticbeanstalk update-environment --application-name "$EB_SERVICE_APP" --environment-name "$EB_SERVICE_ENV" --version-label "$VERSION" --region "$AWS_REGION"
          aws elasticbeanstalk wait environment-updated --application-name "$EB_SERVICE_APP" --environment-names "$EB_SERVICE_ENV" --region "$AWS_REGION"
  migrate:
    stage: post-deploy
    executor: docker
    image: amazon/aws-cli:2.15.0
    steps:
      - uses: flowforge/checkout@v1
      - name: Run migrations and configure environments
        run: |
          # Trigger Scala Service migration endpoint
          SERVICE_CNAME=$(aws elasticbeanstalk describe-environments --application-name "$EB_SERVICE_APP" --environment-names "$EB_SERVICE_ENV" --region "$AWS_REGION" --query "Environments[0].CNAME" --output text)
          curl -s -X POST -H "Authorization: Bearer $SERVICE_ADMIN_TOKEN" "http://${SERVICE_CNAME}/admin/migrate"
          # Run Laravel migrations via SSM
          INSTANCE_ID=$(aws elasticbeanstalk describe-environment-resources --environment-name "$EB_WEB_ENV" --region "$AWS_REGION" --query "EnvironmentResources.Instances[0].Id" --output text)
          aws ssm send-command --instance-ids "$INSTANCE_ID" --document-name "AWS-RunShellScript" --parameters '{"commands":["cd /var/app/current && php artisan migrate --force && php artisan config:cache"]}' --region "$AWS_REGION"
  verify:
    stage: verify
    executor: docker
    image: amazon/aws-cli:2.15.0
    steps:
      - uses: flowforge/checkout@v1
      - name: Verify deployments
        run: |
          WEB_CNAME=$(aws elasticbeanstalk describe-environments --application-name "$EB_WEB_APP" --environment-names "$EB_WEB_ENV" --region "$AWS_REGION" --query "Environments[0].CNAME" --output text)
          SERVICE_CNAME=$(aws elasticbeanstalk describe-environments --application-name "$EB_SERVICE_APP" --environment-names "$EB_SERVICE_ENV" --region "$AWS_REGION" --query "Environments[0].CNAME" --output text)
          echo "Web:     http://${WEB_CNAME}"
          echo "Service: http://${SERVICE_CNAME}"
          curl -sf "http://${WEB_CNAME}/health" && echo " Web OK" || echo " Web FAIL"
          curl -sf "http://${SERVICE_CNAME}/health" && echo " Service OK" || echo " Service FAIL"
`,
	},
}
