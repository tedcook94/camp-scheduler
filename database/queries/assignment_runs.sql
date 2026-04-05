-- name: CreateAssignmentRun :one
INSERT INTO assignment_runs (camp_id, session_id, run_type, status)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, run_type, status, selected_solution_id, created_at;

-- name: GetAssignmentRun :one
SELECT id, camp_id, session_id, run_type, status, selected_solution_id, created_at
FROM assignment_runs
WHERE id = $1 AND camp_id = $2;

-- name: ListAssignmentRunsBySession :many
SELECT id, camp_id, session_id, run_type, status, selected_solution_id, created_at
FROM assignment_runs
WHERE session_id = $1 AND camp_id = $2
ORDER BY created_at DESC;

-- name: UpdateAssignmentRunStatus :one
UPDATE assignment_runs
SET status = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, session_id, run_type, status, selected_solution_id, created_at;

-- name: SelectSolution :one
UPDATE assignment_runs
SET selected_solution_id = $3, status = 'selected'
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, session_id, run_type, status, selected_solution_id, created_at;

-- name: DeleteAssignmentRun :execrows
DELETE FROM assignment_runs
WHERE id = $1 AND camp_id = $2;
