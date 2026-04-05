BEGIN;

ALTER TABLE seasons
    ADD COLUMN start_date date NOT NULL DEFAULT '2000-01-01',
    ADD COLUMN end_date date NOT NULL DEFAULT '2000-12-31';

ALTER TABLE seasons
    ADD CONSTRAINT seasons_date_range CHECK (start_date <= end_date);

ALTER TABLE seasons
    ALTER COLUMN start_date DROP DEFAULT,
    ALTER COLUMN end_date DROP DEFAULT;

COMMIT;
