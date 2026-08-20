# Troubleshooting Guide

This guide covers common issues and their solutions when developing or deploying the Needly Backend.

---

## Table of Contents

- [Server Won't Start](#server-wont-start)
- [Authentication Errors](#authentication-errors)
- [WebSocket Connection Issues](#websocket-connection-issues)
- [Rate Limiting](#rate-limiting)
- [Database Issues](#database-issues)
- [Redis Issues](#redis-issues)
- [Docker Issues](#docker-issues)
- [CI/CD Issues](#cicd-issues)

---

## Server Won't Start

### Port already in use

**Error:**
```
listen tcp :8080: bind: address already in use
```

**Cause:** Another process is already using port 8080.

**Fix:**
```bash
# Find what's using the port
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or change the port in .env
PORT=8081
```

---

### Database connection refused

**Error:**
```
failed to connect to postgres: dial tcp 127.0.0.1:5432: connect: connection refused
```

or:

```
failed to ping database: failed to connect to postgres
```

**Cause:** PostgreSQL is not running or is not reachable at the configured host/port.

**Fix:**
```bash
# Check if PostgreSQL is running
docker compose ps postgres

# Start PostgreSQL if it's down
docker compose up -d postgres

# Verify connectivity
docker compose exec postgres pg_isready -U postgres -d needly
```

Also verify `DB_HOST`, `DB_PORT`, and `DB_PASSWORD` in your `.env` match the running PostgreSQL instance. When using Docker Compose, `DB_HOST` should be `postgres` (the service name), not `localhost`.

---

### Redis connection refused

**Error:**
```
failed to initialize redis: dial tcp 127.0.0.1:6379: connect: connection refused
```

**Cause:** Redis is not running.

**Fix:**
```bash
# Check if Redis is running
docker compose ps redis

# Start Redis
docker compose up -d redis

# Verify connectivity
docker compose exec redis redis-cli ping
# Expected: PONG
```

---

### Default JWT secret in production

**Error:**
```
FATAL: JWT_SECRET is using the default value. Set a strong random secret in production.
```

**Cause:** `GIN_MODE=release` is set but `JWT_SECRET` still has the placeholder value.

**Fix:**
```bash
# Generate a secure secret
openssl rand -hex 32

# Add to .env
JWT_SECRET=<generated-value>
```

---

## Authentication Errors

### Token expired

**Client receives:**
```json
{"error": "invalid or expired token"}
```

**Cause:** The access token has passed its `exp` claim (default: 1 hour).

**Fix:**
Call `POST /api/v1/auth/refresh` with the stored refresh token to obtain a new token pair. See the [API Authentication Guide](api-authentication.md#token-refresh-flow).

---

### Invalid credentials on login

**Client receives:**
```json
{"error": "invalid email or password"}
```

**Cause:** The email or password is incorrect. The error is intentionally generic to prevent user enumeration.

**Fix:**
- Verify the email is registered.
- If the user exists, trigger a password reset flow (not yet implemented) or have them re-register.

---

### Invalid refresh token

**Client receives:**
```json
{"error": "invalid refresh token"}
```

**Cause:** The refresh token hash was not found in the database. The token may have been deleted, never issued, or corrupted.

**Fix:**
Force the user to re-login. The refresh token is no longer valid.

---

### Refresh token family revoked (reuse detection)

**Client receives:**
```json
{"error": "refresh token has been revoked; all sessions in this family terminated"}
```

**Cause:** A previously revoked refresh token was presented. This triggers family-wide revocation as a security measure against token theft.

**Fix:**
Force the user to re-login on all devices. All sessions in the compromised family have been terminated.

---

### Wrong token issuer

**Client receives:**
```
{"error": "invalid token issuer"}
```

**Cause:** The `iss` claim in the JWT does not match the server's `JWT_ISSUER` configuration.

**Fix:**
Ensure `JWT_ISSUER` on the server matches the value used when the token was issued. Tokens issued with a different issuer (e.g., a different environment) will be rejected. Re-issue tokens by having the user log in again.

---

## WebSocket Connection Issues

### CORS rejection

**Browser/mobile client error:**
```
WebSocket connection to 'wss://api.needly.com/api/v1/ws/1' failed
```

**Cause:** The WebSocket origin is not in the allowed CORS list.

**Fix:**
Verify `CORS_ALLOWED_ORIGINS` in `.env` includes the origin of your client application:

```env
CORS_ALLOWED_ORIGINS=https://app.needly.com,https://dev.needly.com
```

For local development, ensure both `http://localhost:3000` and `http://localhost:5173` are included.

---

### WebSocket auth failure

**Client receives (HTTP 401 during handshake):**
```json
{"error": "authorization header is required"}
```

**Cause:** The WebSocket endpoint requires a valid JWT but none was provided.

**Fix:**
Pass the access token as a query parameter when connecting:

```
wss://api.needly.com/api/v1/ws/1?token=<access_token>
```

Or include it in the connection handshake headers if your WebSocket library supports it.

---

### WebSocket household membership required

**Client receives:**
```json
{"error": "not a member of this household"}
```

**Cause:** The authenticated user is not a member of the household they are trying to connect to.

**Fix:**
The user must be added as a household member before they can receive WebSocket events for that household. Use `POST /api/v1/households/:id/members` (household owner only).

---

### WebSocket events not received

**Symptoms:** User connects successfully but does not receive real-time updates.

**Cause:** Multiple possible causes:
1. The user is connected to the wrong household ID.
2. Redis Pub/Sub is down (in multi-instance deployments).
3. The event was published but the user's WebSocket connection dropped silently.

**Fix:**
1. Verify the `household_id` in the WebSocket URL matches the intended household.
2. Check Redis health: `docker compose exec redis redis-cli ping`.
3. Check server logs for WebSocket disconnection messages.
4. Implement client-side heartbeat/ping to detect dead connections.

---

## Rate Limiting

### 429 Too Many Requests

**Client receives:**
```json
{
  "error": "rate limit exceeded",
  "retry_after": 42
}
```

**Response headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
Retry-After: 42
```

**Cause:** The client has exceeded the allowed number of requests (default: 100) within the time window (default: 60 seconds).

**Fix:**
1. Wait `retry_after` seconds before retrying.
2. Implement exponential backoff on the client.
3. Batch or debounce rapid API calls.

**To adjust limits:**
```env
# Increase to 200 requests per 60-second window
RATE_LIMIT_REQUESTS=200

# Or use a wider window (e.g., 5 minutes = 300 seconds)
RATE_LIMIT_WINDOW_SECONDS=300

# Or disable rate limiting entirely (not recommended for production)
RATE_LIMIT_ENABLED=false
```

---

## Database Issues

### Migration failures

**Error (production):**
```
error: relation "users" already exists
```

**Cause:** Migrations have already been applied, or the database schema was created outside of the migration system.

**Fix:**
```bash
# Check current migration version
migrate -path migrations -database "$DATABASE_URL" version

# If the database is ahead or behind, force to the correct version
migrate -path migrations -database "$DATABASE_URL" force <version_number>

# Then apply pending migrations
migrate -path migrations -database "$DATABASE_URL" up
```

---

### Connection pool exhaustion

**Error:**
```
failed to connect to postgres: too many connections open
```

**Cause:** The number of open database connections exceeds PostgreSQL's `max_connections` limit.

**Fix:**
The server uses these pool settings (in `internal/database/postgres.go`):
- `MaxOpenConns`: 25
- `MaxIdleConns`: 5
- `ConnMaxLifetime`: 5 minutes

If you need to increase, modify `internal/database/postgres.go`. Also ensure PostgreSQL's `max_connections` (default: 100) is sufficient for your deployment.

---

### Slow queries in production

**Symptoms:** High latency, request timeouts, memory buildup.

**Fix:**
1. Enable GORM query logging temporarily:
   - The logger level is `Warn` in production and `Info` in debug mode.
   - Switch to `debug` temporarily or add specific query logging.
2. Review the performance indexes in `migrations/000010_add_performance_indexes.sql`.
3. Use `EXPLAIN ANALYZE` on slow queries in PostgreSQL.

---

## Redis Issues

### Redis memory limit reached

**Symptoms:** Cache misses increase, notification history is truncated, rate limiting may malfunction.

**Fix (production compose):**
The production compose already sets:
```yaml
command: redis-server --appendonly yes --maxmemory 128mb --maxmemory-policy allkeys-lru
```

To increase:
```yaml
command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
```

Monitor memory usage:
```bash
docker compose exec redis redis-cli info memory
```

---

### Redis connection refused (Docker)

**Error:**
```
dial tcp 127.0.0.1:6379: connect: connection refused
```

**Cause:** Redis container is not running, or the host/port in `.env` is wrong.

**Fix:**
```bash
# Verify Redis is running
docker compose ps redis

# Restart Redis
docker compose restart redis

# Check Redis logs
docker compose logs redis

# Test connection from the API container
docker compose exec api wget -qO- http://redis:6379 || echo "Redis not reachable"
```

When using Docker Compose, `REDIS_HOST` must be `redis` (the service name), not `localhost`.

---

### Redis Pub/Sub messages not delivered (multi-instance)

**Symptoms:** WebSocket events work on one API instance but not on others.

**Cause:** Redis Pub/Sub is not configured or the `ws:events` channel subscription failed.

**Fix:**
1. Verify `REDIS_HOST` and `REDIS_PASSWORD` are correct on all API instances.
2. Check Redis Pub/Sub channels:
   ```bash
   docker compose exec redis redis-cli pubsub channels
   ```
3. Ensure all instances share the same Redis server.

---

## Docker Issues

### Port conflicts

**Error:**
```
Error starting userland proxy: Bind for 0.0.0.0:8080 failed: port is already allocated
```

**Fix:**
```bash
# Find the process using the port
lsof -i :8080

# Option 1: Stop the conflicting process
kill -9 <PID>

# Option 2: Change the port in docker-compose.yml
ports:
  - "8081:8080"
```

---

### Volume permission denied

**Error:**
```
permission denied while trying to connect to the Docker daemon socket
```

or:
```
cannot create directory: Permission denied
```

**Fix:**
```bash
# Add your user to the docker group (Linux)
sudo usermod -aG docker $USER
# Log out and back in for the change to take effect

# Or for volume permission issues on PostgreSQL
sudo chown -R 1000:1000 ./volumes/postgres_data
```

---

### Container keeps restarting

**Diagnosis:**
```bash
# Check container logs
docker compose logs api --tail=50

# Check container health status
docker compose ps
```

**Common causes:**
1. Database is not ready — ensure `depends_on` health check conditions are met.
2. Missing or incorrect environment variables.
3. `JWT_SECRET` not changed in release mode (fatal error on startup).

---

### Build fails

**Error:**
```
error building image: error from docker: couldn't start build
```

**Fix:**
```bash
# Clean Docker build cache
docker builder prune -a

# Rebuild from scratch
docker compose build --no-cache api
```

---

## CI/CD Issues

### Go version mismatch

**Error:**
```
go: go.mod requires go >= 1.24 (running go 1.22)
```

**Fix:**
Ensure your CI environment uses Go 1.24 or newer. In GitHub Actions:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26'
```

Or update `go.mod` if using a newer Go version locally.

---

### Lint failures

**Error:**
```
golangci-lint: some-hint-that-failed
```

**Fix:**
```bash
# Run linter locally to see full output
make lint

# Auto-fix what's possible
golangci-lint run --fix

# Common fixes:
# - Run gofmt on all files
# - Remove unused imports
# - Add error handling for unchecked errors
```

---

### Test failures in CI

**Error:**
```
FAIL: TestSomething (0.00s)
```

**Fix:**
```bash
# Run tests locally with race detector
make test

# Run with verbose output
go test -v -race ./internal/auth/...

# Run specific test
go test -v -race -run TestSpecificFunction ./internal/...
```

---

## General Debugging Tips

### Enable debug logging

```bash
# Run in debug mode for verbose output
GIN_MODE=debug go run ./cmd/api
```

In debug mode:
- GORM logs all SQL queries.
- Gin logs all request details.
- Auto-migrations run on startup.

### Check all services at once

```bash
# List all containers and their health status
docker compose ps

# Follow all logs
docker compose logs -f

# Check specific service
docker compose logs -f api
```

### Test individual endpoints

```bash
# Health check
curl http://localhost:8080/health

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Authenticated request
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <your-token>"
```

### View Prometheus metrics

```bash
curl http://localhost:8080/metrics
```

### View active WebSocket connections

Check server logs for `websocket client connected` messages, or check the `/metrics` endpoint for WebSocket connection counts.
