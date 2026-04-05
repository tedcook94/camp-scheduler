BEGIN;

CREATE TABLE IF NOT EXISTS assignment_run_selected_solutions (
    camp_id uuid NOT NULL,
    run_id uuid NOT NULL,
    solution_id uuid NOT NULL,
    solution_type text NOT NULL CHECK (solution_type IN ('counselor_cabin', 'camper_cabin')),
    UNIQUE (camp_id, run_id),
    FOREIGN KEY (camp_id, run_id) REFERENCES assignment_runs(camp_id, id) ON DELETE CASCADE
);

-- Migrate existing selected solutions.
INSERT INTO assignment_run_selected_solutions (camp_id, run_id, solution_id, solution_type)
SELECT camp_id, id, selected_solution_id, 'counselor_cabin'
FROM assignment_runs
WHERE selected_solution_id IS NOT NULL;

-- Drop the FK constraints on selected_solution_id before dropping the column.
ALTER TABLE assignment_runs DROP CONSTRAINT IF EXISTS assignment_runs_selected_solution_id_fkey;

ALTER TABLE assignment_runs DROP COLUMN selected_solution_id;

COMMIT;
