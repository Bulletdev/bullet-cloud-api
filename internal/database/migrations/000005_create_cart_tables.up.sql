-- +migrate Up
-- SQL in this section is executed when the migration is applied.

-- Create the carts table (one cart per user)
CREATE TABLE IF NOT EXISTS carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_carts_user
        FOREIGN KEY(user_id) REFERENCES users(id)
        ON DELETE CASCADE -- If user is deleted, their cart is deleted
);

-- Index on user_id for faster lookup
CREATE INDEX IF NOT EXISTS idx_carts_user_id ON carts(user_id);

-- Trigger for updated_at on carts
CREATE TRIGGER update_carts_updated_at
BEFORE UPDATE ON carts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Create the cart_items table
CREATE TABLE IF NOT EXISTS cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id UUID NOT NULL,
    product_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0), -- Price at the time of adding
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cart_items_cart
        FOREIGN KEY(cart_id) REFERENCES carts(id)
        ON DELETE CASCADE, -- If cart is deleted, items are deleted

    CONSTRAINT fk_cart_items_product
        FOREIGN KEY(product_id) REFERENCES products(id)
        ON DELETE CASCADE, -- If product is deleted, remove item from cart

    -- Ensure a product appears only once per cart (update quantity instead)
    CONSTRAINT unique_cart_product UNIQUE (cart_id, product_id)
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);

-- Trigger for updated_at on cart_items
CREATE TRIGGER update_cart_items_updated_at
BEFORE UPDATE ON cart_items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();


-- +migrate Down
-- SQL section moved to the .down.sql file
