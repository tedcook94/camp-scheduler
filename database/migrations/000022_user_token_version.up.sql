ALTER TABLE users ADD COLUMN token_version integer NOT NULL DEFAULT 1;
ALTER TABLE refresh_tokens ADD COLUMN token_version integer NOT NULL DEFAULT 1;
