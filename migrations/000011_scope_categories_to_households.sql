-- +goose Up

-- Add household_id to categories (scoped per household)
ALTER TABLE categories ADD COLUMN household_id BIGINT NOT NULL DEFAULT 0;

-- Drop old global unique index on name
DROP INDEX IF EXISTS idx_categories_name;

-- Create composite unique index: same name allowed in different households
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_household_name
    ON categories (household_id, name);

-- Create index for household lookups
CREATE INDEX IF NOT EXISTS idx_categories_household_id ON categories (household_id);

-- +goose Down
DROP INDEX IF EXISTS idx_categories_household_id;
DROP INDEX IF EXISTS idx_categories_household_name;

-- Restore old global unique index on name
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name ON categories (name);

-- Remove household_id column
ALTER TABLE categories DROP COLUMN IF EXISTS household_id;
