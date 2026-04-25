BEGIN;

ALTER TABLE counselors
    ADD COLUMN first_name text NOT NULL DEFAULT '',
    ADD COLUMN last_name text NOT NULL DEFAULT '';

UPDATE counselors
SET first_name = split_part(counselor_name, ' ', 1),
    last_name = CASE
        WHEN position(' ' in counselor_name) > 0
            THEN trim(substr(counselor_name, position(' ' in counselor_name) + 1))
        ELSE ''
    END;

ALTER TABLE counselors
    ALTER COLUMN first_name DROP DEFAULT,
    ALTER COLUMN last_name DROP DEFAULT;

ALTER TABLE counselors DROP COLUMN counselor_name;

COMMIT;
