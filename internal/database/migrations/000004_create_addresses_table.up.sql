-- +migrate Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    street TEXT NOT NULL,
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    country TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_addresses_user
        FOREIGN KEY(user_id) REFERENCES users(id)
        ON DELETE CASCADE -- Delete addresses if the user is deleted
);

-- Index on the foreign key for performance
CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);

-- Optional: Ensure only one default address per user (more complex, might need a trigger or partial index)
-- Example using a partial unique index (PostgreSQL specific):
-- CREATE UNIQUE INDEX idx_addresses_user_default ON addresses(user_id) WHERE is_default;

-- Trigger for updated_at on addresses
-- Assumes the function update_updated_at_column() exists from migration 000001
CREATE TRIGGER update_addresses_updated_at
BEFORE UPDATE ON addresses
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();


-- +migrate Down
-- SQL section moved to the .down.sql file
