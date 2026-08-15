CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(128) NOT NULL,
    request_method VARCHAR(16) NOT NULL,
    request_path VARCHAR(512) NOT NULL,
    response_status INT NOT NULL,
    response_body BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (key, request_method, request_path)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at
    ON idempotency_keys(created_at);

CREATE TABLE IF NOT EXISTS saves (
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (persona_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_saves_post ON saves(post_id);
CREATE INDEX IF NOT EXISTS idx_saves_created_at ON saves(created_at DESC);

ALTER TABLE media_assets
    ADD COLUMN IF NOT EXISTS file_size BIGINT,
    ADD COLUMN IF NOT EXISTS checksum VARCHAR(128);

CREATE INDEX IF NOT EXISTS idx_media_assets_persona ON media_assets(persona_id);
