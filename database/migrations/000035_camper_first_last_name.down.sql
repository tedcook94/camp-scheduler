BEGIN;

ALTER TABLE campers ADD COLUMN camper_name text NOT NULL DEFAULT '';

UPDATE campers
SET camper_name = trim(both ' ' from first_name || ' ' || last_name);

ALTER TABLE campers ALTER COLUMN camper_name DROP DEFAULT;

ALTER TABLE campers
    DROP COLUMN first_name,
    DROP COLUMN last_name;

COMMIT;
