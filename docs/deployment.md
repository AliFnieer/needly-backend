# Deployment Guide

This guide covers deploying the Needly Backend from local development to production.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development Setup](#local-development-setup)
- [Docker Deployment (Single Instance)](#docker-deployment-single-instance)
- [Docker Compose Production Deployment](#docker-compose-production-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Production Environment Variables](#production-environment-variables)
- [Database Migrations](#database-migrations)
- [Health Check Verification](#health-check-verification)
- [SSL/TLS with Nginx Reverse Proxy](#ssltls-with-nginx-reverse-proxy)

---

## Prerequisites

| Dependency       | Minimum Version | Notes                              |
| ---------------- | --------------- | ---------------------------------- |
| Go               | 1.24+           | Required for local builds          |
| Docker           | 24+             | Required for containerized deploys |
| Docker Compose   | v2+             | For multi-service orchestration    |
| PostgreSQL       | 16+             | Primary database                   |
| Redis            | 7+              | Cache, Pub/Sub, rate limiting      |
| Git              | any             | Source control                     |
| nginx            | 1.24+           | Reverse proxy (production SSL)     |

---

## Local Development Setup

### 1. Clone the repository

```bash
git clone https://github.com/AliFnieer/needly-backend.git
cd needly-backend
```

### 2. Create environment file

```bash
cp .env.example .env
```

Edit `.env` as needed. The defaults work out of the box for local development.

### 3. Start infrastructure services

```bash
docker compose up -d postgres redis
```

This starts PostgreSQL on port `5433` (mapped from container port `5432`) and Redis on port `6379`.

### 4. Run the server

```bash
make run
```

The API starts at `http://localhost:8080`. GORM auto-migrates all models on startup in `debug` mode.

### 5. Seed demo data (optional)

```bash
make seed
```

### 6. Verify

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

---

## Docker Deployment (Single Instance)

Run the full stack (API + PostgreSQL + Redis) in one command:

```bash
docker compose up --build -d
```

This builds the Go binary inside a multi-stage Dockerfile, starts all three services, and auto-migrates on API startup.

### Useful commands

| Command                      | Description                    |
| ---------------------------- | ------------------------------ |
| `docker compose up --build`  | Rebuild and start all services |
| `docker compose down`        | Stop all services              |
| `docker compose logs -f api` | Follow API logs                |
| `docker compose ps`          | List running services          |

### Stopping and removing volumes

```bash
docker compose down -v
```

> **Warning:** This deletes all data in PostgreSQL and Redis volumes.

---

## Docker Compose Production Deployment

A dedicated production compose file is provided at `docker-compose.prod.yml`.

### Start

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

### Production differences from development

| Feature                 | Development (`docker-compose.yml`) | Production (`docker-compose.prod.yml`) |
| ----------------------- | ---------------------------------- | -------------------------------------- |
| `GIN_MODE`              | `debug`                            | `release` (set via env)                |
| API port exposed        | Yes (`8080:8080`)                  | No (internal only, nginx in front)     |
| PostgreSQL port exposed | Yes (`5433:5432`)                  | No (internal network only)             |
| Redis port exposed      | Yes (`6379:6379`)                  | No (internal network only)             |
| Read-only filesystem    | No                                 | Yes (`read_only: true`)                |
| Non-root user           | Built into image                   | Built into image                       |
| Resource limits         | None                               | CPU/memory limits on all containers    |
| Health checks           | Basic                              | Detailed with `start_period`           |
| Security opts          | None                               | `no-new-privileges:true`               |
| Redis eviction          | Default                            | `allkeys-lru` with 128 MB cap          |

### Resource limits (production)

| Container  | CPU Limit | Memory Limit | CPU Reserve | Memory Reserve |
| ---------- | --------- | ------------ | ----------- | -------------- |
| api        | 1.0       | 512 MB       | 0.25        | 128 MB         |
| postgres   | 1.0       | 512 MB       | —           | —              |
| redis      | 0.5       | 128 MB       | —           | —              |

---

## Cloud Deployment

### AWS EC2

1. Launch an `t3.small` (or larger) EC2 instance with Amazon Linux 2023 or Ubuntu 22.04.
2. Install Docker and Docker Compose on the instance.
3. Clone the repository and copy your production `.env` file.
4. Run with the production compose file:

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

5. Open port `80` and `443` in the security group.
6. Set up nginx as a reverse proxy (see [SSL/TLS section](#ssltls-with-nginx-reverse-proxy)).

### AWS ECS (Fargate)

1. Build and push the Docker image to ECR:

```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account>.dkr.ecr.us-east-1.amazonaws.com
docker build -t needly-api .
docker tag needly-api:latest <account>.dkr.ecr.us-east-1.amazonaws.com/needly-api:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/needly-api:latest
```

2. Create ECS Task Definition with the image, environment variables, and port mappings.
3. Use Amazon RDS for PostgreSQL and Amazon ElastiCache for Redis instead of containers.
4. Set `DB_HOST` and `REDIS_HOST` to the managed service endpoints.

### DigitalOcean App Platform

1. Push code to GitHub.
2. Create a new App on DigitalOcean App Platform from the repository.
3. Set environment variables in the app settings.
4. Use a managed PostgreSQL database (DigitalOcean Databases) and managed Redis.
5. The platform builds from the `Dockerfile` automatically.

### Railway

1. Connect your GitHub repository to Railway.
2. Add PostgreSQL and Redis add-ons from the Railway dashboard.
3. Set environment variables, pointing `DB_HOST` and `REDIS_HOST` to the Railway-internal hostnames.
4. Railway deploys automatically on push.

### Fly.io

1. Install the Fly CLI and authenticate:

```bash
fly auth login
```

2. Initialize the app:

```bash
fly launch
```

3. Set secrets:

```bash
fly secrets set JWT_SECRET="your-production-secret" DB_PASSWORD="strong-password" REDIS_PASSWORD="redis-password"
```

4. Deploy:

```bash
fly deploy
```

Use Fly Postgres or an external managed database for production.

---

## Production Environment Variables

The following **must** be changed from their defaults for a production deployment:

| Variable          | Why                                            | Example                                          |
| ----------------- | ---------------------------------------------- | ------------------------------------------------ |
| `GIN_MODE`        | Must be `release` for production               | `release`                                         |
| `JWT_SECRET`      | Default triggers a fatal error in release mode | `a8f7c3e9b2d1...` (64+ hex chars)               |
| `DB_PASSWORD`     | Must not be the default `postgres`             | `xK9#mP2$vL5qR8wN`                              |
| `DB_HOST`         | Must point to the production database          | `db.example.com`                                 |
| `DB_SSLMODE`      | Should be `require` or `verify-full`           | `require`                                         |
| `REDIS_HOST`      | Must point to the production Redis             | `redis.example.com`                              |
| `REDIS_PASSWORD`  | Must be set for production Redis               | `r3d1s-secure-password`                          |
| `CORS_ALLOWED_ORIGINS` | Must list production domains              | `https://app.needly.com`                         |

All other variables have safe defaults but should be reviewed for your deployment.

---

## Database Migrations

### Development

Auto-migrations run on server startup when `GIN_MODE` is not `release`. No manual steps needed.

### Production

Auto-migrations are **skipped** in production mode. Use the `migrate` CLI:

```bash
# Install migrate CLI (if not installed)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply all pending migrations
migrate -path migrations -database "postgres://postgres:PASSWORD@HOST:5432/needly?sslmode=require" up

# Check current migration version
migrate -path migrations -database "postgres://..." version

# Rollback last migration
migrate -path migrations -database "postgres://..." down 1
```

### Migration files

The `migrations/` directory contains numbered SQL files (goose format):

```
000001_create_users.sql
000002_create_households.sql
000003_create_shopping_lists.sql
000004_create_shopping_items.sql
000005_create_categories.sql
000006_add_category_id_to_shopping_items.sql
000007_alter_shopping_items_quantity_decimal.sql
000008_create_shopping_history.sql
000009_create_refresh_tokens.sql
000010_add_performance_indexes.sql
```

### Before deploying a new version

1. Run migrations **before** starting the new API container:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

2. Start the API:

```bash
docker compose -f docker-compose.prod.yml up -d api
```

---

## Health Check Verification

### Endpoint

```text
GET /health
```

This endpoint is public (no authentication required) and returns:

```json
{"status":"ok"}
```

### Verify after deployment

```bash
# From the host
curl -i http://localhost:8080/health

# From outside (with nginx)
curl -i https://api.needly.com/health
```

### Docker health check

The production compose file includes a health check on the API container:

```yaml
healthcheck:
  test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
  interval: 15s
  timeout: 5s
  retries: 3
  start_period: 10s
```

Check container health status:

```bash
docker compose -f docker-compose.prod.yml ps
```

The `api` container should show `healthy` after the start period.

---

## SSL/TLS with Nginx Reverse Proxy

### Install nginx

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install nginx

# Amazon Linux 2023
sudo dnf install nginx
```

### Nginx configuration

Create `/etc/nginx/sites-available/needly`:

```nginx
upstream needly_api {
    server 127.0.0.1:8080;
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name api.needly.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.needly.com;

    # SSL certificates (Let's Encrypt)
    ssl_certificate     /etc/letsencrypt/live/api.needly.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.needly.com/privkey.pem;

    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers (nginx level)
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;

    # Proxy settings
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # API proxy
    location / {
        proxy_pass http://needly_api;
        proxy_read_timeout 30s;
        proxy_send_timeout 30s;
    }

    # WebSocket support
    location /api/v1/ws/ {
        proxy_pass http://needly_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # Health check (can be exposed without proxy auth)
    location /health {
        proxy_pass http://needly_api;
        access_log off;
    }
}
```

### Enable the site

```bash
sudo ln -s /etc/nginx/sites-available/needly /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Obtain SSL certificate with Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d api.needly.com
```

Certbot automatically configures nginx and sets up renewal.

### Verify SSL

```bash
curl -i https://api.needly.com/health
```

Expected: HTTP 200 with `Strict-Transport-Security` header present.

---

## Post-Deployment Checklist

- [ ] `GIN_MODE=release` is set
- [ ] `JWT_SECRET` is a strong random value (not the default)
- [ ] `DB_PASSWORD` is a strong random value
- [ ] `REDIS_PASSWORD` is set
- [ ] Database migrations have been applied
- [ ] `GET /health` returns `{"status":"ok"}`
- [ ] SSL certificate is valid and auto-renewing
- [ ] nginx is proxying WebSocket connections correctly
- [ ] `CORS_ALLOWED_ORIGINS` includes only production domains
- [ ] PostgreSQL and Redis ports are not exposed to the public internet
- [ ] Docker containers are running as non-root
- [ ] Log aggregation is configured (CloudWatch, Datadog, etc.)
