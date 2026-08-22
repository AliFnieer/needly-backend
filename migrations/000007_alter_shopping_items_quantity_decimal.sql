-- +goose Up

-- Alter shopping_items quantity to support decimal values (e.g. 1.5 kg)
ALTER TABLE shopping_items
    ALTER COLUMN quantity TYPE DOUBLE PRECISION;

-- +goose Down
-- Restore integer quantity
ALTER TABLE shopping_items
    ALTER COLUMN quantity TYPE INTEGER;
