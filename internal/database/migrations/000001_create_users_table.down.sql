-- +migrate Down
-- SQL in this section is executed when the migration is rolled back.

-- Drop the trigger first if it exists
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop the function if it exists
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop the index if it exists
DROP INDEX IF EXISTS idx_users_email;

-- Drop the table
DROP TABLE IF EXISTS users;

-- Optional: Drop the extension if it's no longer needed by other tables
-- DROP EXTENSION IF EXISTS "uuid-ossp";
