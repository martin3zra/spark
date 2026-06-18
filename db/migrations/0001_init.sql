-- Base schema for a spark application: users, sessions, preview_cache.

CREATE TABLE IF NOT EXISTS users (
    id                   SERIAL PRIMARY KEY,
    uuid                 UUID NOT NULL DEFAULT gen_random_uuid(),
    name                 TEXT NOT NULL,
    email                TEXT NOT NULL UNIQUE,
    password             TEXT NOT NULL,
    role                 TEXT NOT NULL DEFAULT 'user',
    email_verified_at    TIMESTAMPTZ,
    must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

-- Columns match forge/session DatabaseStore (id, user_id, ip_address,
-- user_agent, payload, created_at, last_activity). created_at/last_activity are
-- Unix-millis (BIGINT), written by the store via time.UnixMilli().
CREATE TABLE IF NOT EXISTS sessions (
    id            TEXT PRIMARY KEY,
    user_id       INTEGER,
    ip_address    TEXT,
    user_agent    TEXT,
    payload       TEXT,
    created_at    BIGINT NOT NULL,
    last_activity BIGINT NOT NULL
);

-- forge/cache PgCache.
CREATE TABLE IF NOT EXISTS preview_cache (
    key     TEXT PRIMARY KEY,
    payload BYTEA NOT NULL
);
