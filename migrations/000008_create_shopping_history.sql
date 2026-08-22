-- +goose Up

-- Create shopping_history table
CREATE TABLE IF NOT EXISTS shopping_history (
    id          BIGSERIAL PRIMARY KEY,
    list_id     BIGINT NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
    item_id     BIGINT REFERENCES shopping_items(id) ON DELETE SET NULL,
    name        VARCHAR(200) NOT NULL,
    quantity    DOUBLE PRECISION NOT NULL DEFAULT 1,
    unit        VARCHAR(50),
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    completed_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_shopping_history_list_id ON shopping_history (list_id);
CREATE INDEX IF NOT EXISTS idx_shopping_history_item_id ON shopping_history (item_id);
CREATE INDEX IF NOT EXISTS idx_shopping_history_completed_at ON shopping_history (completed_at DESC);

-- +goose Down
DROP TABLE IF EXISTS shopping_history;
