# 🥇 Needly Backend

> **Never forget what you need.**

The backend service for **Needly**, a real-time shared shopping and household coordination application.

Needly allows couples, families, and roommates to create shared shopping lists, add and manage items, and synchronize changes between devices in real time.

This repository contains the backend API, business logic, database layer, authentication, caching, and real-time communication infrastructure.

---

## ✨ Features

* 🔐 User registration and authentication with refresh token rotation
* 👥 Household management
* 🛒 Shared shopping lists
* 📝 Shopping item management
* ⚡ Real-time synchronization via WebSockets
* 📜 Shopping history
* 🏷️ Categories, quantities, and units
* 🔒 JWT authentication with refresh token rotation
* 🗄️ PostgreSQL persistence with GORM
* ⚡ Redis for caching and Pub/Sub
* 🛡️ Redis-backed API rate limiting
* 🔔 Push notifications for household changes
* 🛡️ Security headers (HSTS, CSP, X-Frame-Options)
* 📊 Structured logging, metrics, and tracing
* 🐳 Docker-based development
* 🚀 CI/CD with GitHub Actions

---

## 🏗️ Architecture

The backend follows a feature-oriented architecture designed to keep business logic independent from HTTP and infrastructure concerns.

```text
                         ┌─────────────────┐
                         │  Needly Mobile  │
                         │  React Native   │
                         └────────┬────────┘
                                  │
                           HTTPS / WebSocket
                                  │
                                  ▼
                         ┌─────────────────┐
                         │   Needly API    │
                         │   Go + Gin      │
                         └───────┬─┬───────┘
                                 │ │
                   ┌─────────────┘ └─────────────┐
                   │                             │
                   ▼                             ▼
            ┌──────────────┐              ┌──────────────┐
            │  PostgreSQL  │              │    Redis     │
            │              │              │              │
            │ GORM         │              │ Cache /      │
            │ Persistence  │              │ Pub/Sub /    │
            │              │              │ Temporary    │
            └──────────────┘              │ state        │
                                          └──────────────┘
```

### Core domain

```text
User
 │
 │ member of                    belongs to
 ▼                              │
Household                       │
 │                              │
 │ owns                         │
 ▼                              │
Shopping List                   │
 │                              │
 │ contains                     │
 ▼                              │
Shopping Item ──> Shopping History
 │
 └──> Category (optional)
```

A shopping list belongs to a **household**, rather than an individual user. This allows multiple household members to interact with the same lists.

---

## 🛠️ Tech Stack

| Technology     | Purpose                               |
| -------------- | ------------------------------------- |
| Go             | Backend language                      |
| Gin            | HTTP framework                        |
| GORM           | ORM and database access               |
| PostgreSQL     | Primary database                      |
| Redis          | Caching, Pub/Sub, rate limiting       |
| WebSocket      | Real-time client communication        |
| JWT (HS256)    | Authentication with refresh tokens    |
| bcrypt         | Password hashing                      |
| slog (stdlib)  | Structured JSON logging               |
| Docker         | Containerization                      |
| GitHub Actions | CI/CD (lint, test, build, docker)     |

---

## 🗃️ Database

Needly uses **PostgreSQL** as its primary persistent data store.

**GORM** is used as the ORM for:

* Database models
* Queries
* Relationships
* Transactions
* CRUD operations

Example domain relationships:

```text
User
 │
 ├──< HouseholdMember >── Household
 │                           │
 │                           └──< ShoppingList
 │                                    │
 │                                    └──< ShoppingItem
 │                                            │
 │                                            └──< Category (optional)
 │
 ├──< RefreshToken
 │
 ├──< ShoppingHistory
 │
 └──< Notification
```

The database remains the **source of truth** for persistent application data.

Redis is used as a supporting infrastructure component and is not intended to replace PostgreSQL.

---

## ⚡ Redis

Redis provides fast, temporary data storage and messaging capabilities.

Potential uses within Needly include:

### Caching

Frequently accessed data can be cached to reduce unnecessary database queries.

```text
Mobile App
    │
    ▼
Needly API
    │
    ▼
   Redis
    │
    ├── Cache hit → return data
    │
    └── Cache miss
             │
             ▼
        PostgreSQL
```

### Pub/Sub

Redis Pub/Sub distributes WebSocket events between multiple backend instances.

```text
                 ┌─────────────────┐
                 │    Redis        │
                 │    Pub/Sub      │
                 └───────┬─────────┘
                         │
             ┌───────────┼───────────┐
             ▼           ▼           ▼
        API Instance  API Instance  API Instance
             1             2             3
```

When an API instance broadcasts a WebSocket event (e.g. a household broadcast or a client-targeted message), it publishes the event to the `ws:events` Redis channel. Every API instance subscribes to this channel and delivers the event to its locally connected WebSocket clients. Each instance tags its messages with a unique origin ID so it can ignore events it published itself, preventing double delivery.

This enables horizontally scaling the WebSocket infrastructure across multiple API instances.

### Temporary data

Redis is currently used for:

* Rate limiting (fixed-window counters)
* Push notification history (sorted sets per household)
* WebSocket Pub/Sub distribution across API instances

---

## 📁 Project Structure

```text
.
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── seed/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── household/
│   ├── shoppinglist/
│   ├── shoppingitem/
│   ├── category/
│   ├── history/
│   ├── notification/
│   ├── websocket/
│   ├── cache/
│   ├── config/
│   ├── database/
│   ├── docs/
│   ├── middleware/
│   ├── observability/
│   └── server/
│
├── migrations/
├── .github/workflows/
│
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

---

## 🚀 Getting Started

### Prerequisites

Make sure you have:

* Go
* Docker
* Docker Compose
* Git

### Clone

```bash
git clone https://github.com/YOUR_USERNAME/needly-backend.git
cd needly-backend
```

### Environment Variables

Create a `.env` file (see `.env.example` for all options):

```env
PORT=8080
GIN_MODE=debug

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=needly
DB_SSLMODE=disable
DB_TIMEZONE=Africa/Tripoli

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=your_super_secret_jwt_key_change_me
JWT_EXPIRATION_HOURS=1
JWT_REFRESH_TOKEN_TTL_HOURS=720
JWT_ISSUER=needly-api

CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW_SECONDS=60

NOTIFICATIONS_ENABLED=true
NOTIFICATIONS_WEBSOCKET_ENABLED=true
NOTIFICATIONS_HISTORY_LIMIT=50
```

> Never commit real secrets to the repository.

### Start Dependencies

```bash
docker compose up -d
```

This starts:

```text
PostgreSQL
Redis
```

### Run Migrations

The server auto-migrates on startup. For manual migrations, use the [migrate](https://github.com/golang-migrate/migrate) CLI:

```bash
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/needly?sslmode=disable" up
```

### Seed the Database

```bash
make seed
```

### Start the Server

```bash
make run
```

The API will be available at:

```text
http://localhost:8080
```

---

## 🔌 API

The API is versioned under `/api/v1`. Full interactive documentation is available at `/docs` (Swagger UI).

All endpoints except auth endpoints require a valid JWT in the `Authorization: Bearer <token>` header.

### Authentication

```text
POST /api/v1/auth/register          # Create account (returns access + refresh tokens)
POST /api/v1/auth/login             # Log in (returns access + refresh tokens)
POST /api/v1/auth/refresh           # Exchange refresh token for new token pair
GET  /api/v1/auth/me                # Get current user profile
POST /api/v1/auth/logout            # Revoke refresh token(s)
```

### Households

```text
GET    /api/v1/households                        # List user's households
POST   /api/v1/households                        # Create a household
GET    /api/v1/households/:id                    # Get household details
PUT    /api/v1/households/:id                    # Update household (owner only)
DELETE /api/v1/households/:id                    # Delete household (owner only)
POST   /api/v1/households/:id/members            # Add member (owner only)
DELETE /api/v1/households/:id/members/:userId    # Remove member (owner only)
```

### Shopping Lists

```text
GET    /api/v1/households/:id/lists              # List lists for a household
POST   /api/v1/households/:id/lists              # Create a list in a household
GET    /api/v1/lists/:id                         # Get list by ID (includes items)
PUT    /api/v1/lists/:id                         # Update list name
DELETE /api/v1/lists/:id                         # Delete list and its items
```

### Shopping Items

```text
GET    /api/v1/lists/:id/items                   # List items in a shopping list
POST   /api/v1/lists/:id/items                   # Add an item to a list
GET    /api/v1/items/:id                         # Get item by ID
PUT    /api/v1/items/:id                         # Update an item
PATCH  /api/v1/items/:id/completed               # Set completion status
DELETE /api/v1/items/:id                         # Delete an item
POST   /api/v1/history/:id/re-add                # Re-add item from history
```

### Categories

```text
GET    /api/v1/categories                        # List all categories
POST   /api/v1/categories                        # Create a category
GET    /api/v1/categories/:id                    # Get category by ID
PUT    /api/v1/categories/:id                    # Update a category
DELETE /api/v1/categories/:id                    # Delete a category
```

### Shopping History

```text
GET    /api/v1/lists/:id/history                 # History for a specific list
GET    /api/v1/households/:id/history            # History for an entire household
GET    /api/v1/history/:id                       # Get a history entry
DELETE /api/v1/history/:id                       # Delete a history entry
```

Completed items are automatically recorded in history with a snapshot of the item data, who completed it, and when. History entries are preserved even if the original item is deleted.

### Notifications

```text
GET    /api/v1/households/:id/notifications      # Recent notifications for a household
```

### Real-Time

```text
GET    /api/v1/ws/:household_id                  # WebSocket connection (requires auth + membership)
```

Events are pushed to connected clients when shared data changes:

```json
{
  "type": "item.updated",
  "household_id": 1,
  "list_id": 5,
  "item_id": 12,
  "actor_id": 3,
  "title": "Shopping item updated",
  "body": "Item \"Milk\" was updated",
  "created_at": "2026-08-19T12:00:00Z"
}
```

### Observability

```text
GET    /health                                   # Health check (always public)
GET    /metrics                                  # Prometheus-format metrics (auth required in production)
GET    /debug/traces                             # Tracing summary (auth required in production)
GET    /docs                                     # Swagger UI
GET    /docs/openapi.json                        # OpenAPI 3.0.3 spec
```

---

## 🧪 Testing

Run the test suite:

```bash
make test
```

Run tests with coverage:

```bash
make test-cover
```

---

## 🔒 Security

### Authentication

* **JWT access tokens** — Short-lived (configurable, default 1 hour), used for API requests
* **Refresh tokens** — Long-lived (default 30 days), stored as SHA-256 hashes in the database
* **Token rotation** — Each refresh use issues a new token pair and revokes the old one
* **Reuse detection** — If a revoked refresh token is presented, all tokens in its family are revoked
* **Logout** — Revokes specific or all refresh tokens for the user

### Password Security

* Passwords hashed with **bcrypt** (cost factor 10)
* Password hash never serialized in API responses (`json:"-"`)
* Generic error messages prevent user enumeration on login

### Rate Limiting

* Redis-backed fixed-window rate limiter applied globally
* Per-user limits for authenticated requests, per-IP for unauthenticated
* Configurable via environment variables
* Fails open when Redis is unavailable

### Security Headers

Every response includes:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
Content-Security-Policy: default-src 'self'; script-src ...
Strict-Transport-Security: max-age=63072000; includeSubDomains (production only)
```

### WebSocket Security

* Requires JWT authentication
* Verifies household membership before allowing connection
* Origin validation against configured CORS origins

### Input Validation

* All request payloads validated via Gin's struct tag bindings
* Email format, password length, name lengths, quantity ranges enforced
* GORM parameterized queries prevent SQL injection

---

## 📊 Observability

### Structured Logging

All requests are logged in JSON format via Go's `log/slog`:

```json
{"method":"GET","path":"/api/v1/households","status":200,"duration_ms":12,"client_ip":"127.0.0.1","user_id":1,"request_id":"a1b2c3d4-..."}
```

Log level is based on HTTP status: `error` for 5xx, `warn` for 4xx, `info` for success.

### Metrics

In-memory metrics exported in Prometheus text format at `/metrics`:

* HTTP request count, duration, and active connections (per method/path)
* Database query count, duration, and errors
* Cache hit/miss ratio
* WebSocket connections and messages
* Business metrics (users, lists, items)

### Tracing

Lightweight in-memory tracing with request-scoped spans at `/debug/traces`:

* Each request creates a span tracking method, path, duration, and status
* Spans are correlated via the request ID

### Request ID

Every request receives a unique ID (UUIDv4), propagated via the `X-Request-ID` header. Incoming request IDs are honored for distributed tracing.

---

## 🐳 Docker

Start the complete local development environment:

```bash
docker compose up --build
```

Stop the environment:

```bash
docker compose down
```

The local environment consists of:

```text
                    ┌─────────────────┐
                    │   Needly API    │
                    │   Go + Gin      │
                    └───────┬─┬───────┘
                            │ │
                    ┌───────┘ └───────┐
                    ▼                 ▼
             ┌─────────────┐   ┌─────────────┐
             │ PostgreSQL  │   │    Redis    │
             │   + GORM    │   │             │
             └─────────────┘   └─────────────┘
```

---

## 🔄 Real-Time Synchronization

Needly uses WebSockets to synchronize changes between household members.

For example:

```text
User A
  │
  │ checks "Milk"
  ▼
Needly API
  │
  ├── Update PostgreSQL
  │
  └── Publish event
          │
          ▼
        Redis
          │
          ▼
   WebSocket Server
          │
          ▼
       User B
```

User B can see the change immediately without refreshing the application.

---

## 📋 Development Roadmap

### v0.1 — MVP

* [x] User registration
* [x] Login/logout
* [x] Household creation
* [x] Household invitations
* [x] Shopping lists
* [x] Shopping items
* [x] Item completion
* [x] Basic WebSocket synchronization
* [x] PostgreSQL + GORM
* [x] Redis integration

### v0.2

* [x] Categories
* [x] Quantity and units
* [x] Shopping history
* [x] Re-add previous items
* [x] Redis caching
* [x] Improved validation

### v0.3

* [ ] Recurring items
* [x] Push notifications
* [ ] Offline synchronization
* [x] Redis Pub/Sub
* [x] WebSocket horizontal scaling
* [x] Rate limiting

### v1.0

* [x] Production deployment
* [x] CI/CD
* [x] Security review
* [ ] Performance testing
* [x] API documentation
* [x] Observability and monitoring

---

## 🔗 Related Repository

### Needly Mobile

The mobile application is maintained in a separate repository and communicates with this backend through the REST API and WebSocket interface.

---

## 📄 License

This project is licensed under the MIT License.
