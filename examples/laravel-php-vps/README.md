# Laravel PHP — VPS Deployment Example

Full CI/CD pipeline for a Laravel PHP application deployed to a VPS with Nginx.

## Pipeline Stages

| Stage        | Description                                                    |
|--------------|----------------------------------------------------------------|
| **install**  | `composer install` with production optimizations               |
| **test**     | PHPUnit tests, PHP CS Fixer lint, parallel test execution      |
| **package**  | Strip dev files, create timestamped ZIP archive                |
| **deploy**   | SCP archive to VPS, extract to release dir, symlink switch     |
| **post-deploy** | Run `artisan migrate --force`, update `.env`, rebuild caches |
| **cleanup**  | Remove old releases (keep last 5), reload PHP-FPM & Nginx     |

## Directory Layout on VPS

```
/var/www/html/
├── current -> releases/20260327120000   # symlink to active release
├── releases/
│   ├── 20260327120000/                  # latest release
│   ├── 20260326090000/                  # previous release
│   └── ...
└── shared/
    ├── .env                             # persistent environment config
    └── storage/
        └── logs/                        # persistent log directory
```

## Required Secrets / Environment Variables

Configure these in the FlowForge project settings:

| Variable         | Description                          |
|------------------|--------------------------------------|
| `VPS_HOST`       | VPS hostname or IP address           |
| `VPS_USER`       | SSH user for deployment              |
| `VPS_SSH_KEY`    | Private SSH key (PEM format)         |
| `VPS_DEPLOY_PATH`| Deployment root (default `/var/www/html`) |
| `DB_HOST`        | Database host                        |
| `DB_DATABASE`    | Database name                        |
| `DB_USERNAME`    | Database user                        |
| `DB_PASSWORD`    | Database password                    |
| `APP_URL`        | Application URL (e.g. `https://myapp.com`) |

## Nginx Configuration

An example Nginx site config is included in [`nginx/site.conf`](nginx/site.conf). Copy it to `/etc/nginx/sites-available/` on your VPS.

## First-Time VPS Setup

```bash
# On the VPS:
sudo mkdir -p /var/www/html/{releases,shared/storage/logs}
sudo chown -R www-data:www-data /var/www/html

# Copy your .env to the shared directory
sudo cp .env.example /var/www/html/shared/.env
sudo nano /var/www/html/shared/.env   # configure production values

# Install Nginx site config
sudo cp nginx/site.conf /etc/nginx/sites-available/myapp.conf
sudo ln -s /etc/nginx/sites-available/myapp.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```
