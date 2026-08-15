-- Auth migration 002.
-- Adds 'deleted' to the allowed user statuses and indexes grace-period
-- lookups for the account-deletion purge job.

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_status_check;

ALTER TABLE users
    ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'deleting', 'suspended', 'banned', 'deleted'));

CREATE INDEX IF NOT EXISTS idx_users_deletion_grace_period
    ON users (deletion_grace_period_ends_at)
    WHERE status = 'deleting';
