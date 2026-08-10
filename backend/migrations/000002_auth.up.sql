CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'blocked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_sessions_user_id_idx
    ON refresh_sessions (user_id);

CREATE INDEX refresh_sessions_expires_at_idx
    ON refresh_sessions (expires_at);

ALTER TABLE users OWNER TO app_owner;
ALTER TABLE refresh_sessions OWNER TO app_owner;

REVOKE ALL ON TABLE users, refresh_sessions FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users, refresh_sessions TO app_runtime;
