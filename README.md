# 🥇 Needly Backend

> **Never forget what you need.**

The backend service for **Needly**, a real-time shared shopping and household coordination application.

Needly allows couples, families, and roommates to create shared shopping lists, add and manage items, and synchronize changes between devices in real time.

This repository contains the backend API, business logic, database layer, authentication, caching, and real-time communication infrastructure.

---

## ✨ Features

* 🔐 User registration and authentication
* 👥 Household management
* 🛒 Shared shopping lists
* 📝 Shopping item management
* ⚡ Real-time synchronization via WebSockets
* 📜 Shopping history
* 🔄 Recurring shopping items
* 🏷️ Categories, quantities, and units
* 🔒 Authentication and authorization
* 🗄️ PostgreSQL persistence
* ⚡ Redis for caching and real-time infrastructure
* 🐳 Docker-based development
* 🧪 Automated tests
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
            │ Persistent   │              │ Cache /      │
            │ application  │              │ pub/sub /    │
            │ data         │              │ real-time    │
            └──────────────┘              └──────────────┘
```

### Core domain

```text
User
 │
 │ member of
 ▼
Household
 │
 │ owns
 ▼
Shopping List
 │
 │ contains
 ▼
Shopping Item
```

A shopping list belongs to a **household**, rather than an individual user. This allows multiple household members to interact with the same lists.

---

## 🛠️ Tech Stack

| Technology     | Purpose                                        |
| -------------- | ---------------------------------------------- |
| Go             | Backend language                               |
| Gin            | HTTP framework                                 |
| PostgreSQL     | Primary database                               |
| Redis          | Caching, pub/sub, and real-time infrastructure |
| WebSocket      | Real-time client communication                 |
| JWT            | Authentication                                 |
| sqlc           | Type-safe database access                      |
| golang-migrate | Database migrations                            |
| Docker         | Containerization                               |
| GitHub Actions | CI/CD                                          |

---

## ⚡ Redis

Redis is used as a supporting infrastructure component rather than the primary data store.

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

Redis Pub/Sub can be used to distribute events between backend instances.

```text
                    ┌──────────────┐
                    │ PostgreSQL   │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Needly API  │
                    │   Instance 1 │
                    └──────┬───────┘
                           │
                     Redis Pub/Sub
                           │
                    ┌──────┴───────┐
                    │              │
                    ▼              ▼
              API Instance 2   API Instance 3
```

This allows real-time events to continue working when the backend is eventually scaled horizontally.

### Temporary data

Redis can also be used for short-lived data such as:

* Refresh-token metadata
* Rate limiting
* Invitation tokens
* Temporary synchronization state
* WebSocket connection information

The exact Redis responsibilities may evolve as the application grows.

---

## 📁 Project Structure

```text
.
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── household/
│   ├── shoppinglist/
│   ├── shoppingitem/
│   ├── recurring/
│   ├── history/
│   ├── websocket/
│   ├── cache/
│   ├── database/
│   ├── middleware/
│   └── server/
│
├── migrations/
├── sql/
├── tests/
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

### Environment variables

Create a `.env` file:

```env
APP_ENV=development
PORT=8080

DATABASE_URL=postgres://needly:needly@localhost:5432/needly

REDIS_URL=redis://localhost:6379

JWT_SECRET=change-me
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=720h
```

> Never commit real secrets to the repository.

### Start dependencies

```bash
docker compose up -d
```

This starts the local development dependencies:

```text
PostgreSQL
Redis
```

### Run migrations

```bash
make migrate-up
```

### Start the server

```bash
go run ./cmd/server
```

The API will be available at:

```text
http://localhost:8080
```

---

## 🔌 API

The API is versioned under:

```text
/api/v1
```

### Authentication

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
```

### Household

```text
GET  /api/v1/household
POST /api/v1/household/invitations
GET  /api/v1/household/members
```

### Shopping Lists

```text
GET    /api/v1/lists
POST   /api/v1/lists
GET    /api/v1/lists/:id
PATCH  /api/v1/lists/:id
DELETE /api/v1/lists/:id
```

### Shopping Items

```text
GET    /api/v1/lists/:id/items
POST   /api/v1/lists/:id/items
GET    /api/v1/items/:id
PATCH  /api/v1/items/:id
DELETE /api/v1/items/:id
```

### Real-time

```text
GET /api/v1/ws
```

WebSocket events are used to notify connected household members about changes.

Example:

```json
{
  "type": "item.updated",
  "item_id": "item-123"
}
```

---

## 🧪 Testing

Run the test suite with:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

The project aims to maintain strong coverage around business logic, API behavior, authentication, caching, and real-time synchronization.

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
│  Needly API     │
│  Go + Gin       │
└───────┬─────────┘
        │
   ┌────┴────┐
   ▼         ▼
PostgreSQL  Redis
```

---

## 📋 Development Roadmap

### v0.1 — MVP

* [ ] User registration
* [ ] Login/logout
* [ ] Household creation
* [ ] Household invitations
* [ ] Shopping lists
* [ ] Shopping items
* [ ] Item completion
* [ ] Basic WebSocket synchronization
* [ ] Redis integration

### v0.2

* [ ] Categories
* [ ] Quantity and units
* [ ] Shopping history
* [ ] Re-add previous items
* [ ] Redis caching
* [ ] Improved validation

### v0.3

* [ ] Recurring items
* [ ] Push notifications
* [ ] Offline synchronization
* [ ] Redis Pub/Sub
* [ ] WebSocket horizontal scaling
* [ ] Rate limiting

### v1.0

* [ ] Production deployment
* [ ] CI/CD
* [ ] Security review
* [ ] Performance testing
* [ ] API documentation
* [ ] Observability and monitoring

---

## 🔗 Related Repository

Mobile application:

**Needly Mobile**

The mobile application is maintained independently and communicates with this backend through the REST API and WebSocket interface.

---

## 📄 License

This project is licensed under the MIT License.
