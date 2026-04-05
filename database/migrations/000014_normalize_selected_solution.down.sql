BEGIN;

ALTER TABLE assignment_runs ADD COLUMN selected_solution_id uuid;

-- Migrate back only counselor selections; camper selections are discarded
-- because the old schema only supported counselor solution references.
UPDATE assignment_runs ar
SET selected_solution_id = arss.solution_id
FROM assignment_run_selected_solutions arss
WHERE arss.run_id = ar.id AND arss.camp_id = ar.camp_id
  AND arss.solution_type = 'counselor_cabin';

-- Re-add the FK constraint (counselor-only, as it was before).
ALTER TABLE assignment_runs
    ADD CONSTRAINT assignment_runs_selected_solution_id_fkey
        FOREIGN KEY (camp_id, selected_solution_id) REFERENCES counselor_cabin_solutions(camp_id, id) ON DELETE SET NULL;

DROP TABLE IF EXISTS assignment_run_selected_solutions;

COMMIT;
