-- +migrate Up
-- SQL in this section is executed when the migration is applied.

-- Create the categories table
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add index on name for faster lookups
CREATE INDEX IF NOT EXISTS idx_categories_name ON categories(name);

-- Trigger for updated_at on categories
-- Assumes the function update_updated_at_column() exists from migration 000001
CREATE TRIGGER update_categories_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Add the foreign key constraint to the products table
ALTER TABLE products
ADD CONSTRAINT fk_products_category
FOREIGN KEY (category_id) REFERENCES categories(id)
ON DELETE SET NULL;

-- Optional: Add an index on the foreign key column in products for performance
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);


-- +migrate Down
-- SQL section moved to the .down.sql file
