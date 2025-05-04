-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop the trigger first if it exists
DROP TRIGGER IF EXISTS update_products_updated_at ON products;

-- Drop indices if they exist
DROP INDEX IF EXISTS idx_products_name;
-- DROP INDEX IF EXISTS idx_products_category_id;

-- Drop the table
DROP TABLE IF EXISTS products;
