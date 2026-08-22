-- +goose Up
CREATE TABLE idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    route VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    status_code INT NOT NULL,
    content_type VARCHAR(255) NOT NULL DEFAULT 'application/json; charset=utf-8',
    response_body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, route, key_hash)
);

CREATE INDEX idx_idempotency_keys_created_at ON idempotency_keys(created_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
