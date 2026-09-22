-- +goose Up

-- Ordering for household-scoped categories (display order in the category manager)
ALTER TABLE categories ADD COLUMN sort_order BIGINT NOT NULL DEFAULT 0;

-- Backfill existing categories with a stable order so reorder lists are consistent
UPDATE categories AS c
SET sort_order = sub.seq
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY household_id ORDER BY id) AS seq
    FROM categories
) AS sub
WHERE c.id = sub.id;

-- +goose Down
ALTER TABLE categories DROP COLUMN IF EXISTS sort_order;