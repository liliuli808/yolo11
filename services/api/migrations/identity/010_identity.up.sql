CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Identity subsystem migration 010.
-- Adds real-profile metadata to the existing users table and creates the
-- anonymous personas and data-export tables.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS max_personas INTEGER NOT NULL DEFAULT 5 CHECK (max_personas > 0),
    ADD COLUMN IF NOT EXISTS default_persona_id UUID;

CREATE TABLE IF NOT EXISTS personas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    real_profile_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alias TEXT NOT NULL CHECK (length(alias) BETWEEN 1 AND 32),
    bio TEXT CHECK (bio IS NULL OR length(bio) <= 160),
    avatar_seed TEXT NOT NULL CHECK (length(avatar_seed) <= 64),
    avatar_color TEXT NOT NULL CHECK (avatar_color ~ '^#[0-9A-Fa-f]{6}$'),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'restricted', 'archived')),
    is_default BOOLEAN NOT NULL DEFAULT false,
    note_count INTEGER NOT NULL DEFAULT 0 CHECK (note_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_personas_alias_active
    ON personas (alias)
    WHERE status != 'archived';

CREATE UNIQUE INDEX IF NOT EXISTS idx_personas_default
    ON personas (real_profile_id)
    WHERE is_default = true;

CREATE INDEX IF NOT EXISTS idx_personas_real_profile_id
    ON personas (real_profile_id, status);

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS fk_users_default_persona;

ALTER TABLE users
    ADD CONSTRAINT fk_users_default_persona
    FOREIGN KEY (default_persona_id) REFERENCES personas(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS data_exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    real_profile_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'expired')),
    format TEXT NOT NULL CHECK (format IN ('json', 'zip')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ready_at TIMESTAMPTZ,
    download_url TEXT
);

CREATE INDEX IF NOT EXISTS idx_data_exports_real_profile_id
    ON data_exports (real_profile_id, requested_at DESC);
