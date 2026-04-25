BEGIN;

ALTER TABLE counselors ADD COLUMN counselor_name text NOT NULL DEFAULT '';

UPDATE counselors
SET counselor_name = trim(both ' ' from first_name || ' ' || last_name);

ALTER TABLE counselors ALTER COLUMN counselor_name DROP DEFAULT;

ALTER TABLE counselors
    DROP COLUMN first_name,
    DROP COLUMN last_name;

COMMIT;
