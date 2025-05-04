-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop trigger on order_items
DROP TRIGGER IF EXISTS update_order_items_updated_at ON order_items;

-- Drop indices on order_items
DROP INDEX IF EXISTS idx_order_items_product_id;
DROP INDEX IF EXISTS idx_order_items_order_id;

-- Drop the order_items table
DROP TABLE IF EXISTS order_items;

-- Drop trigger on orders
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;

-- Drop indices on orders
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_user_id;

-- Drop the orders table
DROP TABLE IF EXISTS orders;

-- Optional: Drop the ENUM type if it was created
-- DROP TYPE IF EXISTS order_status;
