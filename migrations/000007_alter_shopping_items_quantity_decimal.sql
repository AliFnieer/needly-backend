-- Alter shopping_items quantity to support decimal values (e.g. 1.5 kg)
ALTER TABLE shopping_items
    ALTER COLUMN quantity TYPE DOUBLE PRECISION;