# Docker Multi-Service — Laravel PHP + Golang HTTP Server

Full CI/CD pipeline deploying both a Laravel PHP app and a Golang HTTP server in Docker containers, with MySQL, PostgreSQL, and Redis databases.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Nginx (port 80/443)                │
│              Reverse proxy + static files             │
├──────────────┬───────────────────────────────────────┤
│              │                                       │
│  /           │  /api/*                               │
│  Laravel     │  Golang                               │
│  (PHP-FPM)   │  (HTTP :8080)                         │
│  port 9000   │                                       │
├──────────────┼───────────────────────────────────────┤
│              │                                       │
│  MySQL 8.0   │  PostgreSQL 16                        │
│  (Laravel DB)│  (Golang DB)                          │
│              │                                       │
├──────────────┴───────────────────────────────────────┤
│                    Redis 7                            │
│          (shared cache / sessions / queues)           │
└──────────────────────────────────────────────────────┘
```

## Pipeline Stages

| Stage          | Description                                              |
|----------------|----------------------------------------------------------|
| **test**       | Run PHPUnit tests (Laravel) and `go test` (Golang) in parallel |
| **build-images** | Build Docker images for both services                  |
| **push-images** | Push images to container registry (optional)            |
| **deploy**     | `docker-compose up -d` the full stack                    |
| **post-deploy** | Run Laravel and Go database migrations                  |
| **verify**     | Health check all services (Laravel, Go, MySQL, Postgres, Redis) |

## Network Isolation

| Network           | Services                        | Description                    |
|-------------------|---------------------------------|--------------------------------|
| `frontend`        | nginx, laravel, golang          | Public-facing network          |
| `laravel-backend` | laravel, mysql, redis           | Laravel ↔ MySQL/Redis only     |
| `golang-backend`  | golang, postgres, redis         | Go ↔ Postgres/Redis only       |

MySQL and PostgreSQL are on **internal networks** and are not accessible from outside the Docker stack.

## Persistent Volumes

| Volume          | Mount Path                         | Service    |
|-----------------|------------------------------------|------------|
| `mysql-data`    | `/var/lib/mysql`                   | MySQL      |
| `postgres-data` | `/var/lib/postgresql/data`         | PostgreSQL |
| `redis-data`    | `/data`                            | Redis      |

## Required Environment Variables

| Variable              | Description                      | Default    |
|-----------------------|----------------------------------|------------|
| `COMPOSE_PROJECT`     | Docker Compose project name      | `myapp`    |
| `IMAGE_PREFIX`        | Image name prefix                | `myapp`    |
| `MYSQL_ROOT_PASSWORD` | MySQL root password              | `secret`   |
| `MYSQL_DATABASE`      | Laravel database name            | `laravel`  |
| `MYSQL_USER`          | Laravel database user            | `laravel`  |
| `MYSQL_PASSWORD`      | Laravel database password        | `secret`   |
| `POSTGRES_DB`         | Go database name                 | `goapp`    |
| `POSTGRES_USER`       | Go database user                 | `goapp`    |
| `POSTGRES_PASSWORD`   | Go database password             | `secret`   |
| `REDIS_PASSWORD`      | Redis password                   | `secret`   |
| `REGISTRY_URL`        | Container registry URL           | —          |
| `REGISTRY_USERNAME`   | Registry username                | —          |
| `REGISTRY_PASSWORD`   | Registry password                | —          |

## Project Structure

```
.
├── flowforge.yml                    # CI/CD pipeline
├── docker-compose.yml               # Docker Compose stack
├── nginx/
│   └── nginx.conf                   # Nginx reverse proxy config
├── laravel/
│   ├── Dockerfile                   # Laravel PHP-FPM image
│   ├── composer.json
│   ├── app/
│   └── ...
└── golang/
    ├── Dockerfile                   # Go multi-stage image
    ├── go.mod
    ├── cmd/server/
    ├── frontend/                    # SPA frontend
    ├── migrations/                  # SQL migrations
    └── templates/email/             # Email templates
```

## Quick Start

```bash
# Clone and configure
cp .env.example .env
nano .env

# Start the full stack
docker-compose up -d

# Run migrations
docker-compose exec laravel php artisan migrate --force
docker-compose exec golang ./server migrate up

# Check status
docker-compose ps
```
