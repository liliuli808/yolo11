-- Identity migration 011.
-- Adds optional expiration and updated-at timestamps to data exports and an
-- index for pending-export job queries.

ALTER TABLE data_exports
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

ALTER TABLE data_exports
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_data_exports_pending
    ON data_exports (requested_at)
    WHERE status = 'pending';
