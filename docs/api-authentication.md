# API Authentication Guide

This guide covers the authentication system used by the Needly Backend, designed for mobile app integration.

---

## Table of Contents

- [Overview](#overview)
- [Token Format](#token-format)
- [Registration Flow](#registration-flow)
- [Login Flow](#login-flow)
- [Using Access Tokens](#using-access-tokens)
- [Token Refresh Flow](#token-refresh-flow)
- [Token Rotation](#token-rotation)
- [Logout Flow](#logout-flow)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

---

## Overview

Needly uses a **JWT access token + refresh token** authentication system.

| Token          | Lifetime     | Purpose                                  | Stored in          |
| -------------- | ------------ | ---------------------------------------- | ------------------ |
| Access token   | 1 hour       | Authenticate API requests                | Memory (app state) |
| Refresh token  | 30 days      | Obtain new access/refresh token pairs    | Secure storage     |

- **Access tokens** are short-lived JWTs signed with HS256. They are sent with every API request.
- **Refresh tokens** are cryptographically random 64-character hex strings. They are stored as SHA-256 hashes in the database and are never sent back in responses.
- **Token rotation** means every refresh operation issues a new token pair and revokes the old one.
- **Reuse detection** revokes the entire token family if a previously revoked refresh token is presented.

### Base URL

```
https://api.needly.com/api/v1
```

---

## Token Format

### Access Token (JWT) Claims

```json
{
  "user_id": 1,
  "email": "user@example.com",
  "iss": "needly-api",
  "exp": 1724088000,
  "iat": 1724084400,
  "jti": "a1b2c3d4e5f6..."
}
```

| Claim     | Description                                    |
| --------- | ---------------------------------------------- |
| `user_id` | Integer ID of the authenticated user           |
| `email`   | User's email address                           |
| `iss`     | Token issuer (default: `needly-api`)           |
| `exp`     | Expiration timestamp (Unix)                    |
| `iat`     | Issued-at timestamp (Unix)                     |
| `jti`     | Unique token ID (random hex, 32 characters)    |

### Auth Response Object

All auth endpoints return:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "first_name": "John",
    "last_name": "Doe",
    "email": "user@example.com",
    "created_at": "2026-08-19T10:00:00Z",
    "updated_at": "2026-08-19T10:00:00Z"
  }
}
```

| Field           | Type    | Description                                  |
| --------------- | ------- | -------------------------------------------- |
| `access_token`  | string  | JWT to include in `Authorization` header     |
| `refresh_token` | string  | Token used to obtain a new token pair        |
| `token_type`    | string  | Always `"Bearer"`                            |
| `expires_in`    | integer | Access token lifetime in seconds (3600)      |
| `user`          | object  | The authenticated user's profile             |

---

## Registration Flow

### Endpoint

```
POST /api/v1/auth/register
```

### Request

```bash
curl -X POST https://api.needly.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

### Request body

| Field        | Type   | Required | Constraints        | Description        |
| ------------ | ------ | -------- | ------------------ | ------------------ |
| `first_name` | string | yes      | min 2, max 100     | User's first name  |
| `last_name`  | string | yes      | min 2, max 100     | User's last name   |
| `email`      | string | yes      | valid email format | User's email       |
| `password`   | string | yes      | min 8, max 72      | Account password   |

### Response (201 Created)

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "f1e2d3c4b5a6...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "created_at": "2026-08-19T10:00:00Z",
    "updated_at": "2026-08-19T10:00:00Z"
  }
}
```

### Error responses

| Status | Error                            | Cause                              |
| ------ | -------------------------------- | ---------------------------------- |
| 400    | `email is required`              | Missing or invalid email field     |
| 400    | `password must be at least 8`    | Password too short                 |
| 409    | `email already registered`       | Email exists in the database       |

---

## Login Flow

### Endpoint

```
POST /api/v1/auth/login
```

### Request

```bash
curl -X POST https://api.needly.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

### Request body

| Field      | Type   | Required | Description     |
| ---------- | ------ | -------- | --------------- |
| `email`    | string | yes      | User's email    |
| `password` | string | yes      | Account password |

### Response (200 OK)

Returns the same `AuthResponse` object as registration.

### Error responses

| Status | Error                         | Cause                           |
| ------ | ----------------------------- | ------------------------------- |
| 400    | `email is required`           | Missing or invalid email field  |
| 401    | `invalid email or password`   | Wrong credentials               |

> **Note:** The login error message is intentionally generic to prevent user enumeration.

---

## Using Access Tokens

Include the access token in the `Authorization` header for all authenticated requests:

```bash
curl https://api.needly.com/api/v1/auth/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Header format

```
Authorization: Bearer <access_token>
```

### What the server validates

1. The token is a valid HS256 JWT signed with the server's `JWT_SECRET`.
2. The `exp` claim has not passed.
3. The `iss` claim matches the configured issuer (default: `needly-api`).

### Error responses

| Status | Error                          | Cause                        |
| ------ | ------------------------------ | ---------------------------- |
| 401    | `authorization header is required` | No `Authorization` header  |
| 401    | `invalid authorization header format` | Malformed header        |
| 401    | `invalid or expired token`     | Token is expired or corrupt  |
| 401    | `invalid token issuer`         | `iss` claim doesn't match    |

---

## Token Refresh Flow

When the access token expires (or is about to), use the refresh token to get a new pair.

### When to refresh

- When you receive a `401` response with `invalid or expired token`.
- Proactively, a few minutes before `expires_in` elapses.

### Endpoint

```
POST /api/v1/auth/refresh
```

### Request

```bash
curl -X POST https://api.needly.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "f1e2d3c4b5a6..."
  }'
```

### Request body

| Field           | Type   | Required | Description         |
| --------------- | ------ | -------- | ------------------- |
| `refresh_token` | string | yes      | Current refresh token |

### Response (200 OK)

Returns a **new** `AuthResponse` with a new access token and a new refresh token. The old refresh token is revoked.

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...(new)",
  "refresh_token": "new_refresh_token_value...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": { ... }
}
```

### Error responses

| Status | Error                                        | Cause                                |
| ------ | -------------------------------------------- | ------------------------------------ |
| 401    | `invalid refresh token`                      | Token hash not found in database     |
| 401    | `refresh token has been revoked; all sessions in this family terminated` | Token reuse detected |
| 401    | `refresh token has expired`                  | Refresh token TTL exceeded           |

---

## Token Rotation

Every refresh operation implements **token rotation with reuse detection**:

1. Client calls `/auth/refresh` with refresh token `RT-1`.
2. Server revokes `RT-1` and issues a new pair: access token `AT-2` + refresh token `RT-2`.
3. Both `RT-1` and `RT-2` belong to the same **family** (a shared `family_id`).
4. If `RT-1` is presented again after revocation, the **entire family is revoked** — this detects token theft or replay.

### Lifecycle diagram

```
Register/Login
    │
    ▼
  RT-1 ──────► Refresh ──────► RT-2 ──────► Refresh ──────► RT-3
    │            │               │            │
    │            │ revokes       │            │ revokes
    │            ▼               │            ▼
    │          RT-1              │          RT-2
    │          (revoked)         │          (revoked)
    │                            │
    └── same family_id ──────────┘
```

---

## Logout Flow

### Endpoint

```
POST /api/v1/auth/logout
```

Requires authentication (access token in `Authorization` header).

### Logout from current session

Sends the current refresh token to revoke only that session:

```bash
curl -X POST https://api.needly.com/api/v1/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "f1e2d3c4b5a6..."
  }'
```

### Logout from all sessions

Send an empty body or omit the `refresh_token` field:

```bash
curl -X POST https://api.needly.com/api/v1/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Response (200 OK)

```json
{
  "message": "logged out successfully"
}
```

---

## Error Handling

### Error response format

All errors follow a consistent format:

```json
{
  "error": "human-readable error message"
}
```

### Common errors and recommended client actions

| HTTP Status | Error Message         | Client Action                                       |
| ----------- | --------------------- | --------------------------------------------------- |
| 400         | Validation errors     | Fix the request body and retry                      |
| 401         | `invalid or expired token` | Call `/auth/refresh` with the refresh token       |
| 401         | `invalid refresh token` | Force re-login; refresh token is invalid            |
| 401         | `refresh token has been revoked; all sessions in this family terminated` | Force re-login; possible token theft |
| 401         | `refresh token has expired` | Force re-login; refresh token TTL exceeded        |
| 409         | `email already registered` | Prompt user to log in or use a different email   |
| 429         | `rate limit exceeded` | Wait `retry_after` seconds, then retry              |
| 500         | `internal server error` | Retry with exponential backoff; report if persistent |

### Handling 401 errors — recommended flow

```
API request fails with 401
    │
    ├── Error is "invalid or expired token"
    │       │
    │       ├── Refresh token available?
    │       │       │
    │       │       ├── YES → Call /auth/refresh
    │       │       │         │
    │       │       │         ├── Success → Retry original request with new access token
    │       │       │         │
    │       │       │         └── Failure (401) → Force re-login
    │       │       │
    │       │       └── NO → Force re-login
    │       │
    │       └── Retry original request only ONCE to avoid infinite loops
    │
    └── Error is "invalid refresh token" or "revoked" → Force re-login
```

---

## Best Practices

### Storing tokens

| Token          | Storage                  | Notes                                         |
| -------------- | ------------------------ | --------------------------------------------- |
| Access token   | In-memory / app state    | Never persist to disk; lost on app restart    |
| Refresh token  | iOS: Keychain            | Use platform secure storage                   |
|                | Android: EncryptedSharedPreferences | Use platform secure storage            |
|                | Never in: localStorage, plain text files, logs |                         |

### Handling concurrent requests

When multiple API requests run simultaneously and the access token expires:

1. **Queue** — Hold all failed requests until the first refresh completes.
2. **Single-flight** — Use a mutex or flag to ensure only one refresh call runs at a time.
3. **Retry** — After refresh succeeds, retry all queued requests with the new access token.

Do **not** fire multiple refresh requests in parallel — only the first succeeds; the rest will fail with reuse detection.

### Proactive token refresh

Refresh the access token before it expires rather than waiting for a 401:

```typescript
// Pseudocode
const EXPIRY_BUFFER_SECONDS = 60; // refresh 1 minute before expiry

function getAccessToken(): string {
  const now = Math.floor(Date.now() / 1000);
  if (now >= tokenExpiry - EXPIRY_BUFFER_SECONDS) {
    // Token about to expire — refresh now
    scheduleTokenRefresh();
  }
  return currentAccessToken;
}
```

### Security checklist

- [ ] Refresh tokens stored in platform-secure storage only
- [ ] Access tokens held in memory, not persisted
- [ ] Single-flight refresh to prevent race conditions
- [ ] Exponential backoff on retry after refresh failure
- [ ] Force re-login when refresh token is revoked or expired
- [ ] Clear all tokens on logout
- [ ] Never log or transmit refresh tokens in plaintext
