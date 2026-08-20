# Contract Tests

Contract tests validate that the API implementation matches the OpenAPI specification (`internal/docs/openapi.json`). They ensure:

1. **Spec coverage** — all expected endpoints and status codes exist in the OpenAPI spec.
2. **Runtime correctness** — the actual HTTP responses match the contract (status codes, response shapes).

## What is tested

| Endpoint | Contract Point |
|---|---|
| `GET /health` | 200, response has `status` field |
| `POST /api/v1/auth/register` | 201 on success, 409 on duplicate email |
| `POST /api/v1/auth/login` | 200 on success, 401 on bad credentials |
| `GET /api/v1/households` | 200, response is an array |
| `GET /api/v1/households/:id/categories` | 200, response is an array |
| `POST /api/v1/households/:id/categories` | 201 on success |

## Running

```bash
go test ./internal/contract/ -v
```

## How it works

1. The OpenAPI spec is loaded from the embedded `openapi.json` at init time.
2. Static tests verify the spec contains the expected paths, operations, and status codes.
3. Runtime tests spin up a Gin test server with an in-memory SQLite database (via `testutil.SetupTestDB`) and issue real HTTP requests, asserting on status codes and response shapes.
