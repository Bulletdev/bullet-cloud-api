-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop trigger on cart_items
DROP TRIGGER IF EXISTS update_cart_items_updated_at ON cart_items;

-- Drop indices on cart_items
DROP INDEX IF EXISTS idx_cart_items_product_id;
DROP INDEX IF EXISTS idx_cart_items_cart_id;

-- Drop the cart_items table (constraints/FKs are dropped with the table)
DROP TABLE IF EXISTS cart_items;

-- Drop trigger on carts
DROP TRIGGER IF EXISTS update_carts_updated_at ON carts;

-- Drop index on carts
DROP INDEX IF EXISTS idx_carts_user_id;

-- Drop the carts table
DROP TABLE IF EXISTS carts;
