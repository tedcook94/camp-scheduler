BEGIN;

ALTER TABLE assignment_runs
    DROP COLUMN IF EXISTS is_stale;

COMMIT;
