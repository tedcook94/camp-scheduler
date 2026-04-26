-- Restore legacy authentication tables. Note: data is not recoverable, this
-- only re-creates the schema so an environment can roll back to the prior
-- application version.

CREATE EXTENSION IF NOT EXISTS pg_cron;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    camp_id uuid REFERENCES camps(id) ON DELETE CASCADE,
    username text UNIQUE NOT NULL,
    email text UNIQUE NOT NULL,
    password_hash text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    role text NOT NULL DEFAULT 'admin',
    token_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'super_admin')),
    CONSTRAINT users_camp_id_role_check CHECK (
        (role = 'super_admin' AND camp_id IS NULL) OR
        (role <> 'super_admin' AND camp_id IS NOT NULL)
    )
);

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    token_version integer NOT NULL DEFAULT 1,
    impersonated_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;
