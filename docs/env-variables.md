# Environment Variables Reference

All configuration is loaded from environment variables at startup. Values can also be placed in a `.env` file in the project root (loaded automatically via `godotenv`).

---

## Server

| Variable   | Type   | Default    | Required | Description                                                      | Example        |
| ---------- | ------ | ---------- | -------- | ---------------------------------------------------------------- | -------------- |
| `PORT`     | string | `8080`     | no       | Port the HTTP server listens on                                   | `8080`         |
| `GIN_MODE` | string | `debug`    | no       | Gin framework mode. Use `release` for production. Triggers auto-migration skip and HSTS headers. | `release` |

---

## Database

| Variable       | Type   | Default       | Required | Description                                                          | Example                           |
| -------------- | ------ | ------------- | -------- | -------------------------------------------------------------------- | --------------------------------- |
| `DB_HOST`      | string | `localhost`   | no       | PostgreSQL host                                                       | `db.example.com`                  |
| `DB_PORT`      | string | `5432`        | no       | PostgreSQL port                                                       | `5432`                            |
| `DB_USER`      | string | `postgres`    | no       | PostgreSQL user                                                       | `needly_app`                      |
| `DB_PASSWORD`  | string | `postgres`    | **yes (prod)** | PostgreSQL password. Default is only for local development.     | `xK9#mP2$vL5qR8wN`               |
| `DB_NAME`      | string | `needly`      | no       | PostgreSQL database name                                              | `needly`                          |
| `DB_SSLMODE`   | string | `disable`     | no       | PostgreSQL SSL mode. Use `require` or `verify-full` in production.   | `require`                         |
| `DB_TIMEZONE`  | string | `Africa/Tripoli` | no    | Timezone used by the database connection                              | `UTC`                             |

---

## Redis

| Variable        | Type   | Default     | Required | Description                                                              | Example          |
| --------------- | ------ | ----------- | -------- | ------------------------------------------------------------------------ | ---------------- |
| `REDIS_HOST`    | string | `localhost` | no       | Redis host                                                               | `redis.example.com` |
| `REDIS_PORT`    | string | `6379`      | no       | Redis port                                                               | `6379`           |
| `REDIS_PASSWORD`| string | `""`        | no       | Redis password. Empty string means no authentication. Required in prod.  | `r3d1s-pass`     |
| `REDIS_DB`      | int    | `0`         | no       | Redis database number (0-15)                                             | `0`              |

---

## JWT

| Variable                  | Type   | Default                            | Required | Description                                                              | Example                    |
| ------------------------- | ------ | ---------------------------------- | -------- | ------------------------------------------------------------------------ | -------------------------- |
| `JWT_SECRET`              | string | `your_super_secret_jwt_key_change_me` | **yes (prod)** | Secret key used to sign JWT tokens with HS256. Fatal error in release mode if unchanged. | `a8f7c3e9b2d1...` (64+ hex chars) |
| `JWT_EXPIRATION_HOURS`    | int    | `1`                                | no       | Access token lifetime in hours                                            | `1`                        |
| `JWT_REFRESH_TOKEN_TTL_HOURS` | int | `720`                            | no       | Refresh token lifetime in hours (720 = 30 days)                           | `720`                      |
| `JWT_ISSUER`              | string | `needly-api`                       | no       | Value of the `iss` claim in JWTs. Clients should not need to change this. | `needly-api`               |

---

## CORS

| Variable              | Type        | Default                            | Required | Description                                                          | Example                              |
| --------------------- | ----------- | ---------------------------------- | -------- | -------------------------------------------------------------------- | ------------------------------------ |
| `CORS_ALLOWED_ORIGINS`| `[]string`  | `http://localhost:3000,http://localhost:5173` | no | Comma-separated list of allowed origins for CORS. Set to production domain(s) in prod. | `https://app.needly.com` |

---

## Rate Limiting

Redis-backed fixed-window rate limiter. Applied globally to all API endpoints with stricter limits for auth endpoints.

| Variable                    | Type | Default | Required | Description                                                                 | Example |
| --------------------------- | ---- | ------- | -------- | --------------------------------------------------------------------------- | ------- |
| `RATE_LIMIT_ENABLED`       | bool | `true`  | no       | Enable or disable rate limiting entirely.                                    | `true`  |
| `RATE_LIMIT_REQUESTS`      | int  | `100`   | no       | Maximum number of requests per window for general API endpoints.            | `100`   |
| `RATE_LIMIT_WINDOW_SECONDS`| int  | `60`    | no       | Rate limit window duration in seconds.                                       | `60`    |

### How rate limiting works

- **Authenticated users**: keyed by `user:<id>` — each user gets their own limit.
- **Unauthenticated requests**: keyed by `ip:<client_ip>` — per-IP limit.
- **Auth endpoints** (`/auth/register`, `/auth/login`, `/auth/refresh`): limited to **10 requests per minute** (hardcoded strict limit).
- **Redis failure**: the limiter **fails closed** — returns 503 if Redis is unavailable.
- **Response headers**: every response includes `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `Retry-After` (when limited).

---

## Notifications

Controls push notification delivery and WebSocket notification behavior.

| Variable                       | Type | Default | Required | Description                                                                    | Example |
| ------------------------------ | ---- | ------- | -------- | ------------------------------------------------------------------------------ | ------- |
| `NOTIFICATIONS_ENABLED`        | bool | `true`  | no       | Enable push notification system.                                               | `true`  |
| `NOTIFICATIONS_WEBSOCKET_ENABLED` | bool | `true` | no    | Enable WebSocket-based real-time notification delivery.                        | `true`  |
| `NOTIFICATIONS_HISTORY_LIMIT`  | int  | `50`    | no       | Maximum number of notification history entries kept per household in Redis.    | `50`    |

---

## Distributed Tracing (OpenTelemetry)

The application uses OpenTelemetry with OTLP HTTP export. Traces are sent to a collector endpoint (e.g. Jaeger, Grafana Tempo).

| Variable                      | Type   | Default      | Required | Description                                          | Example              |
| ----------------------------- | ------ | ------------ | -------- | ---------------------------------------------------- | -------------------- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | `""`         | no       | OTLP HTTP collector endpoint (host:port). Empty disables tracing. | `jaeger:4318`        |
| `OTEL_SERVICE_NAME`           | string | `needly-api` | no       | Service name shown in trace backends.                | `needly-api`         |
| `LOG_LEVEL`                   | string | `info`       | no       | Structured log level: debug, info, warn, error.      | `info`               |

### Metrics

The `/metrics` endpoint exposes Prometheus-format metrics (auth-protected). Key metric families:

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `http_requests_total` | counter | method, path, status | Total HTTP requests |
| `http_request_duration_seconds` | histogram | method, path, status | Request latency |
| `http_active_requests` | gauge | — | In-flight requests |
| `db_queries_total` | counter | operation | DB query count |
| `db_query_duration_seconds` | histogram | operation | DB query latency |
| `db_pool_open_connections` | gauge | — | Open DB connections |
| `cache_hits_total` / `cache_misses_total` | counter | — | Cache hit/miss count |
| `ws_active_connections` | gauge | — | Active WebSocket connections |

---

## TLS (Optional)

When both variables are set, the server starts with HTTPS using `ListenAndServeTLS`. When empty, plain HTTP is used.

| Variable         | Type   | Default | Required | Description                                  | Example                    |
| ---------------- | ------ | ------- | -------- | -------------------------------------------- | -------------------------- |
| `TLS_CERT_FILE`  | string | `""`    | no       | Path to the TLS certificate file (PEM).      | `/etc/letsencrypt/live/api.needly.com/fullchain.pem` |
| `TLS_KEY_FILE`   | string | `""`    | no       | Path to the TLS private key file (PEM).      | `/etc/letsencrypt/live/api.needly.com/privkey.pem`   |

---

## Database Connection Pool

These variables control the database connection pool for production tuning.

| Variable                        | Type | Default | Required | Description                                              | Example |
| ------------------------------- | ---- | ------- | -------- | -------------------------------------------------------- | ------- |
| `DB_MAX_OPEN_CONNS`            | int  | `25`    | no       | Maximum number of open database connections.              | `25`    |
| `DB_MAX_IDLE_CONNS`            | int  | `5`     | no       | Maximum number of idle database connections.              | `5`     |
| `DB_CONN_MAX_LIFETIME_MINUTES` | int  | `5`     | no       | Maximum lifetime of a connection in minutes.              | `5`     |

---

## Quick Reference: Production `.env`

Copy and fill in the values for a production deployment:

```env
# Server
PORT=8080
GIN_MODE=release

# Database
DB_HOST=your-production-db-host
DB_PORT=5432
DB_USER=needly_app
DB_PASSWORD=CHANGE_ME_STRONG_PASSWORD
DB_NAME=needly
DB_SSLMODE=require
DB_TIMEZONE=UTC

# Redis
REDIS_HOST=your-production-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=CHANGE_ME_REDIS_PASSWORD
REDIS_DB=0

# JWT
JWT_SECRET=GENERATE_64_HEX_CHARS_HERE
JWT_EXPIRATION_HOURS=1
JWT_REFRESH_TOKEN_TTL_HOURS=720
JWT_ISSUER=needly-api

# CORS
CORS_ALLOWED_ORIGINS=https://app.needly.com

# Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW_SECONDS=60

# Notifications
NOTIFICATIONS_ENABLED=true
NOTIFICATIONS_WEBSOCKET_ENABLED=true
NOTIFICATIONS_HISTORY_LIMIT=50

# TLS (optional — use when not behind nginx/ALB)
# TLS_CERT_FILE=/etc/letsencrypt/live/api.needly.com/fullchain.pem
# TLS_KEY_FILE=/etc/letsencrypt/live/api.needly.com/privkey.pem

# Database connection pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MINUTES=5

# OpenTelemetry tracing
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4318
OTEL_SERVICE_NAME=needly-api
LOG_LEVEL=info
```

### Generating a secure JWT secret

```bash
openssl rand -hex 32
```

This produces a 64-character hex string suitable for `JWT_SECRET`.

---

## Defaults Summary

| Variable                       | Default Value                           |
| ------------------------------ | --------------------------------------- |
| `PORT`                         | `8080`                                  |
| `GIN_MODE`                     | `debug`                                 |
| `DB_HOST`                      | `localhost`                             |
| `DB_PORT`                      | `5432`                                  |
| `DB_USER`                      | `postgres`                              |
| `DB_PASSWORD`                  | `postgres`                              |
| `DB_NAME`                      | `needly`                                |
| `DB_SSLMODE`                   | `disable`                               |
| `DB_TIMEZONE`                  | `Africa/Tripoli`                        |
| `REDIS_HOST`                   | `localhost`                             |
| `REDIS_PORT`                   | `6379`                                  |
| `REDIS_PASSWORD`               | `""` (empty)                            |
| `REDIS_DB`                     | `0`                                     |
| `JWT_SECRET`                   | `your_super_secret_jwt_key_change_me`   |
| `JWT_EXPIRATION_HOURS`         | `1`                                     |
| `JWT_REFRESH_TOKEN_TTL_HOURS`  | `720`                                   |
| `JWT_ISSUER`                   | `needly-api`                            |
| `CORS_ALLOWED_ORIGINS`         | `http://localhost:3000,http://localhost:5173` |
| `RATE_LIMIT_ENABLED`           | `true`                                  |
| `RATE_LIMIT_REQUESTS`          | `100`                                   |
| `RATE_LIMIT_WINDOW_SECONDS`    | `60`                                    |
| `NOTIFICATIONS_ENABLED`        | `true`                                  |
| `NOTIFICATIONS_WEBSOCKET_ENABLED` | `true`                               |
| `NOTIFICATIONS_HISTORY_LIMIT`  | `50`                                    |
| `TLS_CERT_FILE`               | `""` (empty, plain HTTP)                |
| `TLS_KEY_FILE`                | `""` (empty, plain HTTP)                |
| `DB_MAX_OPEN_CONNS`           | `25`                                    |
| `DB_MAX_IDLE_CONNS`           | `5`                                     |
| `DB_CONN_MAX_LIFETIME_MINUTES`| `5`                                     |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `""` (disabled)                          |
| `OTEL_SERVICE_NAME`           | `needly-api`                             |
| `LOG_LEVEL`                   | `info`                                   |
