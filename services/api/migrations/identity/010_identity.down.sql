-- Rollback identity subsystem migration 010.

ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_default_persona;

DROP TABLE IF EXISTS data_exports;
DROP TABLE IF EXISTS personas;

ALTER TABLE users DROP COLUMN IF EXISTS default_persona_id;
ALTER TABLE users DROP COLUMN IF EXISTS max_personas;
