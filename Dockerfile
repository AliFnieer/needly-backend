# ---- Build Stage ----
FROM golang:1.26.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with version info via ldflags
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /needly-api ./cmd/api

# ---- Runtime Stage ----
FROM alpine:3.21

# Install runtime dependencies and timezone data
RUN apk add --no-cache ca-certificates tzdata curl \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy the built binary
COPY --from=builder /needly-api .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Create non-root user with a fixed UID
RUN adduser -D -H -u 10001 appuser
USER appuser

EXPOSE 8080

# Health check: hits the /health endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Version label
ARG VERSION=dev
LABEL version="${VERSION}"

CMD ["./needly-api"]
