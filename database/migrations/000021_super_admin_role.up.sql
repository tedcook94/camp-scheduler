BEGIN;

ALTER TABLE users ALTER COLUMN camp_id DROP NOT NULL;

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'super_admin'));

ALTER TABLE users ADD CONSTRAINT users_camp_id_role_check
    CHECK ((role = 'super_admin' AND camp_id IS NULL) OR (role != 'super_admin' AND camp_id IS NOT NULL));

COMMIT;
