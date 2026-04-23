BEGIN;

-- Drop all legacy counselor_cabin and camper_cabin runs. The combined
-- cabin run replaces both; pre-prod, no data is preserved.
DELETE FROM assignment_runs
    WHERE run_type IN ('counselor_cabin', 'camper_cabin');

-- Restrict run_type to the new combined cabin run plus activity_schedule.
ALTER TABLE assignment_runs
    ADD CONSTRAINT assignment_runs_run_type_check
    CHECK (run_type IN ('cabin', 'activity_schedule'));

-- Enforce one run per (session, run_type). Triggers must DELETE-then-INSERT
-- inside a transaction; old runs of the same type are replaced.
ALTER TABLE assignment_runs
    ADD CONSTRAINT assignment_runs_session_type_unique
    UNIQUE (session_id, run_type);

-- Update the selected-solution check to include the new cabin type and the
-- previously-omitted activity_schedule. The old types are no longer reachable
-- because the runs that referenced them have been deleted.
ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS assignment_run_selected_solutions_solution_type_check;
ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS arss_solution_type_check;
ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT assignment_run_selected_solutions_solution_type_check
    CHECK (solution_type IN ('cabin', 'activity_schedule'));

COMMIT;
