BEGIN;

ALTER TABLE assignment_runs
    ADD COLUMN is_stale boolean NOT NULL DEFAULT false;

COMMIT;
