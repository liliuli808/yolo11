ALTER TABLE users ALTER COLUMN email_normalized SET NOT NULL;
DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
ALTER TABLE users DROP COLUMN IF EXISTS username;
