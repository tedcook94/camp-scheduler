ALTER TABLE refresh_tokens ADD COLUMN impersonated_by uuid REFERENCES users(id) ON DELETE SET NULL;
