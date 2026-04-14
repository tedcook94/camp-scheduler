BEGIN;

DELETE FROM refresh_tokens
WHERE user_id IN (SELECT id FROM users WHERE role = 'super_admin');

DELETE FROM users WHERE role = 'super_admin';

ALTER TABLE users DROP CONSTRAINT users_camp_id_role_check;

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin'));

ALTER TABLE users ALTER COLUMN camp_id SET NOT NULL;

COMMIT;
