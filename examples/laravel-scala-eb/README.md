# Laravel/PHP + Scala Service — Elastic Beanstalk Deployment

Full CI/CD pipeline for a multi-tier application with a Laravel PHP web frontend, a Scala WAR backend service, and a shared Scala API Interface library. Deployed to AWS Elastic Beanstalk.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        AWS Elastic Beanstalk                        │
│                                                                     │
│  ┌─────────────────────────┐   ┌──────────────────────────────┐    │
│  │  Environment A (Web)    │   │  Environment B (Service)     │    │
│  │  PHP Platform           │   │  Tomcat Platform             │    │
│  │                         │   │                              │    │
│  │  ┌───────────────────┐  │   │  ┌────────────────────────┐ │    │
│  │  │ Laravel PHP App   │  │──>│  │ Scala WAR Service      │ │    │
│  │  │ (Web Frontend)    │  │   │  │ (Business Logic API)   │ │    │
│  │  └───────────────────┘  │   │  └────────────────────────┘ │    │
│  │          │               │   │          │                  │    │
│  │          v               │   │          v                  │    │
│  │  ┌───────────────────┐  │   │  ┌────────────────────────┐ │    │
│  │  │ RDS MySQL         │  │   │  │ RDS PostgreSQL         │ │    │
│  │  └───────────────────┘  │   │  └────────────────────────┘ │    │
│  └─────────────────────────┘   └──────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Build Dependency Chain

The pipeline respects a strict build order:

```
                    ┌─────────────────────────┐
                    │  1. Scala API Interface  │
                    │     sbt package → JAR    │
                    └───────────┬─────────────┘
                                │
                    ┌───────────v─────────────┐
                    │  2. Scala WAR Service   │
                    │  API JAR → lib/         │
                    │  sbt package → WAR      │
                    └───────────┬─────────────┘
                                │
               ┌────────────────┼────────────────┐
               │                                 │
   ┌───────────v───────────┐         ┌───────────v───────────┐
   │  3. Laravel Web       │         │  4. Package for EB    │
   │  composer install     │         │  WAR → ROOT.war       │
   │  test → ZIP           │         │  + .ebextensions      │
   └───────────┬───────────┘         └───────────┬───────────┘
               │                                 │
               v                                 v
   ┌───────────────────────┐         ┌───────────────────────┐
   │  5. Deploy Web to EB  │         │  6. Deploy Service    │
   │  (PHP Platform)       │         │  to EB (Tomcat)       │
   └───────────┬───────────┘         └───────────┬───────────┘
               │                                 │
               └─────────────┬───────────────────┘
                             │
               ┌─────────────v─────────────┐
               │  7. Post-Deploy           │
               │  - Migrate databases      │
               │  - Update env vars        │
               │  - Verify health          │
               └───────────────────────────┘
```

## Project Structure

```
.
├── flowforge.yml                           # CI/CD pipeline
├── scala-api-interface/                    # Shared Scala library (JAR)
│   ├── build.sbt
│   ├── project/
│   │   ├── build.properties
│   │   └── plugins.sbt
│   └── src/main/scala/com/example/api/
│       ├── ApiClient.scala
│       ├── models/
│       │   ├── Request.scala
│       │   └── Response.scala
│       └── interfaces/
│           ├── UserService.scala
│           └── OrderService.scala
├── scala-service/                          # Scala WAR Service (Tomcat)
│   ├── build.sbt
│   ├── project/
│   │   ├── build.properties
│   │   └── plugins.sbt
│   ├── lib/                                # API Interface JAR placed here by CI
│   └── src/
│       ├── main/
│       │   ├── scala/com/example/service/
│       │   │   ├── ServiceServlet.scala
│       │   │   ├── impl/
│       │   │   │   ├── UserServiceImpl.scala
│       │   │   │   └── OrderServiceImpl.scala
│       │   │   └── db/
│       │   │       └── DatabaseMigrator.scala
│       │   ├── resources/
│       │   │   └── application.conf
│       │   └── webapp/
│       │       └── WEB-INF/
│       │           └── web.xml
│       └── test/scala/
└── web/                                    # Laravel PHP Web Application
    ├── composer.json
    ├── composer.lock
    ├── .env.example
    ├── app/
    ├── config/
    ├── database/migrations/
    ├── routes/
    ├── resources/
    └── public/
```

## Pipeline Stages

| Stage                | Description                                                        |
|----------------------|--------------------------------------------------------------------|
| **build-api-interface** | Compile + test + package Scala API Interface as JAR             |
| **build-scala-service** | Download API JAR → `lib/`, compile + test + package as WAR     |
| **build-web**        | `composer install`, `php artisan test`, create deployment ZIP       |
| **package**          | Upload ZIP + WAR bundles to S3 for EB deployment                   |
| **deploy-web**       | Create EB version + update PHP environment                         |
| **deploy-service**   | Create EB version + update Tomcat environment                      |
| **post-deploy**      | Run migrations, update env vars for both, cross-link services      |
| **verify**           | Health check both environments, report status                      |

## Required Secrets / Environment Variables

### AWS Credentials

| Variable                | Description                               |
|-------------------------|-------------------------------------------|
| `AWS_ACCESS_KEY_ID`     | AWS IAM access key                        |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM secret key                        |
| `AWS_REGION`            | AWS region (default `us-east-1`)          |

### Elastic Beanstalk

| Variable              | Description                                 |
|-----------------------|---------------------------------------------|
| `EB_WEB_APP`          | EB application name for Laravel Web         |
| `EB_WEB_ENV`          | EB environment name for Laravel Web         |
| `EB_SERVICE_APP`      | EB application name for Scala Service       |
| `EB_SERVICE_ENV`      | EB environment name for Scala Service       |
| `S3_BUCKET`           | S3 bucket for deployment artifacts          |

### Database

| Variable              | Description                                 |
|-----------------------|---------------------------------------------|
| `DB_HOST`             | RDS endpoint for Laravel (MySQL)            |
| `DB_DATABASE`         | Laravel database name                       |
| `DB_USERNAME`         | Laravel database user                       |
| `DB_PASSWORD`         | Laravel database password                   |
| `SERVICE_DB_HOST`     | RDS endpoint for Scala Service (PostgreSQL) |
| `SERVICE_DB_DATABASE` | Service database name                       |
| `SERVICE_DB_USERNAME` | Service database user                       |
| `SERVICE_DB_PASSWORD` | Service database password                   |
| `SERVICE_ADMIN_TOKEN` | Admin token for triggering Service migrations |

## Elastic Beanstalk Platform Requirements

### Web Environment (Laravel)
- **Platform**: PHP 8.3 running on 64bit Amazon Linux 2023
- **Instance type**: t3.medium or higher
- **RDS**: MySQL 8.0

### Service Environment (Scala)
- **Platform**: Tomcat 10 with Corretto 17 running on 64bit Amazon Linux 2023
- **Instance type**: t3.large (JVM needs more memory)
- **RDS**: PostgreSQL 16
- **JVM Options**: `-Xms512m -Xmx1024m`

## EB Extensions

The pipeline auto-generates `.ebextensions` configs for:
- JVM memory settings for Tomcat
- Environment variable injection
- Application-specific configuration

Custom `.ebextensions` can also be placed in:
- `web/.ebextensions/` — for Laravel
- `scala-service/.ebextensions/` — for Scala Service

## Quick Start

```bash
# 1. Set up AWS credentials
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
export AWS_REGION=us-east-1

# 2. Create EB applications and environments
eb init my-web-app --platform "PHP 8.3" --region us-east-1
eb create web-prod

eb init my-service-app --platform "Tomcat 10 with Corretto 17" --region us-east-1
eb create service-prod

# 3. Create S3 bucket for artifacts
aws s3 mb s3://my-deploy-artifacts

# 4. Configure FlowForge project with the environment variables above

# 5. Push to main branch — the pipeline handles the rest
git push origin main
```
