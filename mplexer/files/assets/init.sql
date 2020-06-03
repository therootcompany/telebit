CREATE extension IF NOT EXISTS pgcrypto;
SET TIMEZONE='UTC';

--DROP TABLE IF EXISTS authorizations;
CREATE TABLE IF NOT EXISTS authorizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    shared_key TEXT NOT NULL,
    public_key TEXT NOT NULL,
    machine_ppid TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'UTC'),
    deleted_at TIMESTAMP NOT NULL DEFAULT ('epoch' AT TIME ZONE 'UTC')
);

--CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_slug ON authorizations (slug);
CREATE INDEX IF NOT EXISTS idx_slug ON authorizations (slug);
CREATE INDEX IF NOT EXISTS idx_machine_ppid ON authorizations (machine_ppid);
CREATE INDEX IF NOT EXISTS idx_public_key ON authorizations (public_key);

