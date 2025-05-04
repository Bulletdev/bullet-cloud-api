-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop the foreign key index on products first
DROP INDEX IF EXISTS idx_products_category_id;

-- Drop the foreign key constraint from the products table
ALTER TABLE products
DROP CONSTRAINT IF EXISTS fk_products_category;

-- Drop the trigger on categories
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;

-- Drop the index on categories name
DROP INDEX IF EXISTS idx_categories_name;

-- Drop the categories table
DROP TABLE IF EXISTS categories;
