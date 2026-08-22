-- +goose Up

-- Recurrence rule for shopping items ('', 'daily', 'weekly', 'biweekly', 'monthly')
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS recurrence_rule VARCHAR(20) NOT NULL DEFAULT '';

-- When a completed recurring item becomes due again (NULL = not scheduled)
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS next_due_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_shopping_items_next_due_at ON shopping_items (next_due_at);

-- +goose Down
DROP INDEX IF EXISTS idx_shopping_items_next_due_at;

ALTER TABLE shopping_items DROP COLUMN IF EXISTS next_due_at;

ALTER TABLE shopping_items DROP COLUMN IF EXISTS recurrence_rule;
