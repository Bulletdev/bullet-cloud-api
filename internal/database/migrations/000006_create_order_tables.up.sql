-- +migrate Up
-- SQL in this section is executed when the migration is applied.

-- Define allowed order statuses (if not using ENUM type)
-- Consider using an ENUM type in PostgreSQL for better type safety:
-- CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');

-- Create the orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    shipping_address_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled')),
    total NUMERIC(10, 2) NOT NULL CHECK (total >= 0),
    tracking_number TEXT NULL, -- Nullable tracking number
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_orders_user
        FOREIGN KEY(user_id) REFERENCES users(id)
        ON DELETE CASCADE, -- If user is deleted, their orders are deleted

    CONSTRAINT fk_orders_address
        FOREIGN KEY(shipping_address_id) REFERENCES addresses(id)
        ON DELETE SET NULL -- Keep order history even if address is deleted
);

-- Indices for faster querying
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- Trigger for updated_at on orders
CREATE TRIGGER update_orders_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Create the order_items table
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0), -- Price at the time of order
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_items_order
        FOREIGN KEY(order_id) REFERENCES orders(id)
        ON DELETE CASCADE, -- If order is deleted, items are deleted

    CONSTRAINT fk_order_items_product
        FOREIGN KEY(product_id) REFERENCES products(id)
        ON DELETE RESTRICT -- Prevent deleting a product that is part of an order
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- Trigger for updated_at on order_items
CREATE TRIGGER update_order_items_updated_at
BEFORE UPDATE ON order_items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();


-- +migrate Down
-- SQL section moved to the .down.sql file
