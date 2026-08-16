-- Create shopping_lists table
CREATE TABLE IF NOT EXISTS shopping_lists (
    id           BIGSERIAL PRIMARY KEY,
    household_id BIGINT NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name         VARCHAR(150) NOT NULL,
    created_by   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_shopping_lists_household_id ON shopping_lists (household_id);