-- name: SelectSolution :one
INSERT INTO assignment_run_selected_solutions (camp_id, run_id, solution_id, solution_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (camp_id, run_id) DO UPDATE
SET solution_id = EXCLUDED.solution_id, solution_type = EXCLUDED.solution_type
RETURNING camp_id, run_id, solution_id, solution_type;

-- name: GetSelectedSolution :one
SELECT camp_id, run_id, solution_id, solution_type
FROM assignment_run_selected_solutions
WHERE run_id = $1 AND camp_id = $2;

-- name: DeselectSolution :execrows
DELETE FROM assignment_run_selected_solutions
WHERE run_id = $1 AND camp_id = $2;
