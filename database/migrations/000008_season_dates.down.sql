BEGIN;

ALTER TABLE seasons DROP CONSTRAINT seasons_date_range;
ALTER TABLE seasons DROP COLUMN start_date, DROP COLUMN end_date;

COMMIT;
