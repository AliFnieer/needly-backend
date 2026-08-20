-- +goose Up

-- Household membership lookups
CREATE INDEX IF NOT EXISTS idx_household_members_user_id ON household_members(user_id);
CREATE INDEX IF NOT EXISTS idx_household_members_household_id ON household_members(household_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_household_members_unique ON household_members(household_id, user_id);

-- Shopping list lookups by household
CREATE INDEX IF NOT EXISTS idx_shopping_lists_household_id ON shopping_lists(household_id);

-- Shopping item lookups by list
CREATE INDEX IF NOT EXISTS idx_shopping_items_list_id ON shopping_items(list_id);
CREATE INDEX IF NOT EXISTS idx_shopping_items_category_id ON shopping_items(category_id);
CREATE INDEX IF NOT EXISTS idx_shopping_items_completed ON shopping_items(list_id, is_completed);

-- History lookups
CREATE INDEX IF NOT EXISTS idx_shopping_history_list_id ON shopping_history(list_id);
CREATE INDEX IF NOT EXISTS idx_shopping_history_completed_at ON shopping_history(completed_at DESC);

-- Refresh token lookups
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_shopping_history_completed_at;
DROP INDEX IF EXISTS idx_shopping_history_list_id;
DROP INDEX IF EXISTS idx_shopping_items_completed;
DROP INDEX IF EXISTS idx_shopping_items_category_id;
DROP INDEX IF EXISTS idx_shopping_items_list_id;
DROP INDEX IF EXISTS idx_shopping_lists_household_id;
DROP INDEX IF EXISTS idx_household_members_unique;
DROP INDEX IF EXISTS idx_household_members_household_id;
DROP INDEX IF EXISTS idx_household_members_user_id;
