BEGIN;

DROP TABLE IF EXISTS activity_explanations;
DROP TABLE IF EXISTS activity_assignments;
DROP TABLE IF EXISTS activity_solutions;

-- Restore the original solution_type check without activity_schedule.
ALTER TABLE assignment_run_selected_solutions
    DROP CONSTRAINT IF EXISTS arss_solution_type_check;

ALTER TABLE assignment_run_selected_solutions
    ADD CONSTRAINT arss_solution_type_check
    CHECK (solution_type IN ('counselor_cabin', 'camper_cabin'));

COMMIT;
