-- +goose Up

-- Add category_id column to shopping_items table
ALTER TABLE shopping_items
    ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL;

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_shopping_items_category_id ON shopping_items (category_id);

-- +goose Down
ALTER TABLE shopping_items DROP COLUMN IF EXISTS category_id;
