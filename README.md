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
* 🗄️ PostgreSQL persistence with GORM
* ⚡ Redis for caching and Pub/Sub
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

| Technology     | Purpose                               |
| -------------- | ------------------------------------- |
| Go             | Backend language                      |
| Gin            | HTTP framework                        |
| GORM           | ORM and database access               |
| PostgreSQL     | Primary database                      |
| Redis          | Caching, Pub/Sub, and temporary state |
| WebSocket      | Real-time client communication        |
| JWT            | Authentication                        |
| golang-migrate | Database migrations                   |
| Docker         | Containerization                      |
| GitHub Actions | CI/CD                                 |

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
 │
 └──< ItemHistory
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

Redis Pub/Sub can distribute events between multiple backend instances.

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

This provides a path toward horizontally scaling the WebSocket infrastructure.

### Temporary data

Redis may also be used for short-lived data such as:

* Rate limiting
* Invitation tokens
* Temporary synchronization state
* WebSocket connection metadata
* Short-lived caches

Redis responsibilities will evolve as the application grows.

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
│   │
│   ├── database/
│   │   ├── models/
│   │   ├── postgres.go
│   │   └── repositories/
│   │
│   ├── middleware/
│   └── server/
│
├── migrations/
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

### Environment Variables

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

```bash
make migrate-up
```

### Start the Server

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

### Categories

```text
GET    /api/v1/categories
POST   /api/v1/categories
GET    /api/v1/categories/:id
PUT    /api/v1/categories/:id
DELETE /api/v1/categories/:id
```

### Shopping Items

```text
GET    /api/v1/lists/:id/items
POST   /api/v1/lists/:id/items
GET    /api/v1/items/:id
PATCH  /api/v1/items/:id
PUT    /api/v1/items/:id
DELETE /api/v1/items/:id
```

Shopping items support an optional `category_id` field that links to a category:

```json
{
  "name": "Milk",
  "quantity": 2,
  "unit": "liters",
  "category_id": 3
}
```

### Real-Time

```text
GET /api/v1/ws
```

WebSocket events notify connected household members when shared data changes.

Example event:

```json
{
  "type": "item.updated",
  "item_id": "item-123"
}
```

---

## 🧪 Testing

Run the test suite:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

The project aims to maintain strong test coverage around:

* Business logic
* API behavior
* Authentication
* Database operations
* Redis functionality
* WebSocket communication
* Authorization
* Error handling

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

* [ ] User registration
* [ ] Login/logout
* [ ] Household creation
* [ ] Household invitations
* [ ] Shopping lists
* [ ] Shopping items
* [ ] Item completion
* [ ] Basic WebSocket synchronization
* [ ] PostgreSQL + GORM
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

### Needly Mobile

The mobile application is maintained in a separate repository and communicates with this backend through the REST API and WebSocket interface.

---

## 📄 License

This project is licensed under the MIT License.
