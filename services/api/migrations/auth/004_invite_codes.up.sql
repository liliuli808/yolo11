CREATE TABLE IF NOT EXISTS invite_codes (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code       text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    used_by    uuid REFERENCES users(id),
    used_at    timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_invite_codes_code ON invite_codes (code);