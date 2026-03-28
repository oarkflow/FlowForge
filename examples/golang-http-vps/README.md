# Golang HTTP Server — VPS Deployment Example

Full CI/CD pipeline for a Go HTTP server with a frontend SPA, deployed to a VPS behind Nginx.

## Pipeline Stages

| Stage             | Description                                                              |
|-------------------|--------------------------------------------------------------------------|
| **build-frontend** | `npm ci && npm run build` — produces `frontend/dist/`                   |
| **prepare**       | Move frontend dist, migration files, email templates to Go project root  |
| **test**          | `go vet`, `go test -race` with coverage                                  |
| **build**         | `go build` with ldflags (version, build time), produces binary           |
| **package**       | Create `.tar.gz` with binary, static, migrations, templates              |
| **deploy**        | SCP to VPS, extract to release dir, copy static to Nginx, symlink switch |
| **post-deploy**   | Run DB migrations, update `.env`, restart service                        |
| **cleanup**       | Remove old releases (keep last 5), reload Nginx                          |

## Project Structure

```
.
├── cmd/server/          # Go main package
├── internal/            # Go internal packages
├── frontend/            # SPA frontend (React/Vue/Svelte)
│   ├── src/
│   ├── package.json
│   └── dist/            # built by CI
├── migrations/          # SQL migration files
│   ├── 001_init.up.sql
│   └── 001_init.down.sql
├── templates/
│   └── email/           # HTML email templates
│       ├── welcome.html
│       └── reset.html
├── go.mod
├── go.sum
└── flowforge.yml
```

## Directory Layout on VPS

```
/opt/goapp/
├── current -> releases/20260327120000    # symlink to active release
├── releases/
│   ├── 20260327120000/
│   │   ├── server                        # Go binary
│   │   ├── static/                       # frontend assets
│   │   ├── migrations/                   # SQL migrations
│   │   ├── templates/email/              # email templates
│   │   └── .env -> ../../shared/.env
│   └── ...
└── shared/
    └── .env                              # persistent env config

/var/www/goapp/
└── static/                               # Nginx-served static files
```

## Required Secrets / Environment Variables

| Variable         | Description                            |
|------------------|----------------------------------------|
| `VPS_HOST`       | VPS hostname or IP address             |
| `VPS_USER`       | SSH user for deployment                |
| `VPS_SSH_KEY`    | Private SSH key (PEM format)           |
| `VPS_DEPLOY_PATH`| Deployment root (default `/opt/goapp`) |
| `DB_HOST`        | Database host                          |
| `DB_DATABASE`    | Database name                          |
| `DB_USERNAME`    | Database user                          |
| `DB_PASSWORD`    | Database password                      |
| `SMTP_HOST`      | SMTP server for email templates        |
| `SMTP_PORT`      | SMTP port                              |

## Nginx & Systemd Configuration

- [`nginx/site.conf`](nginx/site.conf) — Nginx reverse proxy + static file serving
- [`systemd/goapp.service`](systemd/goapp.service) — systemd unit for the Go binary

## First-Time VPS Setup

```bash
# Create service user
sudo useradd -r -s /bin/false goapp

# Create directory structure
sudo mkdir -p /opt/goapp/{releases,shared}
sudo mkdir -p /var/www/goapp/static
sudo chown -R goapp:goapp /opt/goapp
sudo chown -R www-data:www-data /var/www/goapp

# Create initial .env
sudo cp .env.example /opt/goapp/shared/.env
sudo nano /opt/goapp/shared/.env

# Install systemd service
sudo cp systemd/goapp.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable goapp

# Install Nginx config
sudo cp nginx/site.conf /etc/nginx/sites-available/goapp.conf
sudo ln -s /etc/nginx/sites-available/goapp.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```
