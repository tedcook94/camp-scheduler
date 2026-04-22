BEGIN;

ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS arss_unique_per_session_type;

ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS arss_camp_session_fkey;

ALTER TABLE assignment_run_selected_solutions
    DROP COLUMN IF EXISTS session_id;

COMMIT;
