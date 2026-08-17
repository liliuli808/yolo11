ALTER TABLE users ADD COLUMN username TEXT;
ALTER TABLE users ADD COLUMN password_hash TEXT;

CREATE UNIQUE INDEX idx_users_username ON users (username);

ALTER TABLE users ALTER COLUMN email_normalized DROP NOT NULL;
