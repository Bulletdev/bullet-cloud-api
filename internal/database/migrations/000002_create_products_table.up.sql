-- +migrate Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    -- Use NUMERIC for monetary values, e.g., NUMERIC(precision, scale)
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    category_id UUID NULL, -- Initially nullable, FK constraint added later
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL -- Add later when categories table exists
);

-- Optional: Add indices for frequently queried columns
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
-- CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id); -- Add later

-- Trigger to automatically update updated_at timestamp
-- Assumes the function update_updated_at_column() was created in migration 000001
CREATE TRIGGER update_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();


-- +migrate Down
-- SQL section moved to the .down.sql file
