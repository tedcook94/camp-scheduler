BEGIN;

ALTER TABLE campers
    ADD COLUMN first_name text NOT NULL DEFAULT '',
    ADD COLUMN last_name text NOT NULL DEFAULT '';

UPDATE campers
SET first_name = split_part(camper_name, ' ', 1),
    last_name = CASE
        WHEN position(' ' in camper_name) > 0
            THEN trim(substr(camper_name, position(' ' in camper_name) + 1))
        ELSE ''
    END;

ALTER TABLE campers
    ALTER COLUMN first_name DROP DEFAULT,
    ALTER COLUMN last_name DROP DEFAULT;

ALTER TABLE campers DROP COLUMN camper_name;

COMMIT;
