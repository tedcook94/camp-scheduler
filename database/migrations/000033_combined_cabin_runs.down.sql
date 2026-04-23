BEGIN;

ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS assignment_run_selected_solutions_solution_type_check;
ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT arss_solution_type_check
    CHECK (solution_type IN ('counselor_cabin', 'camper_cabin', 'activity_schedule'));

ALTER TABLE assignment_runs
    DROP CONSTRAINT IF EXISTS assignment_runs_session_type_unique;
ALTER TABLE assignment_runs
    DROP CONSTRAINT IF EXISTS assignment_runs_run_type_check;

COMMIT;
