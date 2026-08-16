-- Create shopping_items table
CREATE TABLE IF NOT EXISTS shopping_items (
    id           BIGSERIAL PRIMARY KEY,
    list_id      BIGINT NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
    name         VARCHAR(200) NOT NULL,
    quantity     INTEGER NOT NULL DEFAULT 1,
    unit         VARCHAR(50),
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_by   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_shopping_items_list_id ON shopping_items (list_id);