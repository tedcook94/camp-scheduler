-- name: SelectSolution :one
INSERT INTO assignment_run_selected_solutions (camp_id, session_id, run_id, solution_id, solution_type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (camp_id, run_id) DO UPDATE
SET solution_id = EXCLUDED.solution_id, solution_type = EXCLUDED.solution_type
RETURNING camp_id, session_id, run_id, solution_id, solution_type;

-- name: GetSelectedSolution :one
SELECT camp_id, session_id, run_id, solution_id, solution_type
FROM assignment_run_selected_solutions
WHERE run_id = $1 AND camp_id = $2;

-- name: DeselectSolution :execrows
DELETE FROM assignment_run_selected_solutions
WHERE run_id = $1 AND camp_id = $2;

-- name: ListConflictingSelectedRuns :many
SELECT ar.id
FROM assignment_runs ar
JOIN assignment_run_selected_solutions arss
    ON arss.run_id = ar.id AND arss.camp_id = ar.camp_id
WHERE ar.camp_id = $1
    AND ar.session_id = $2
    AND ar.run_type = $3
    AND ar.id <> $4;
