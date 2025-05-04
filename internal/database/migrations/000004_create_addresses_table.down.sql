-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop the trigger first
DROP TRIGGER IF EXISTS update_addresses_updated_at ON addresses;

-- Drop the index on user_id
DROP INDEX IF EXISTS idx_addresses_user_id;

-- Drop the partial unique index if it was created
-- DROP INDEX IF EXISTS idx_addresses_user_default;

-- Drop the addresses table (FK constraint is dropped automatically with the table)
DROP TABLE IF EXISTS addresses;
