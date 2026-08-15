DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS saves;

ALTER TABLE media_assets
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS checksum;
