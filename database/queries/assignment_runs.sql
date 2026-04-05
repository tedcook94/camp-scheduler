-- name: CreateAssignmentRun :one
INSERT INTO assignment_runs (camp_id, session_id, run_type, status)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, run_type, status, created_at;

-- name: GetAssignmentRun :one
SELECT id, camp_id, session_id, run_type, status, created_at
FROM assignment_runs
WHERE id = $1 AND camp_id = $2;

-- name: ListAssignmentRunsBySession :many
SELECT
    ar.id,
    ar.camp_id,
    ar.session_id,
    ar.run_type,
    ar.status,
    ar.created_at,
    arss.solution_id AS selected_solution_id
FROM assignment_runs ar
LEFT JOIN assignment_run_selected_solutions arss ON arss.run_id = ar.id AND arss.camp_id = ar.camp_id
WHERE ar.session_id = $1 AND ar.camp_id = $2
ORDER BY ar.created_at DESC;

-- name: UpdateAssignmentRunStatus :one
UPDATE assignment_runs
SET status = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, session_id, run_type, status, created_at;

-- name: DeleteAssignmentRun :execrows
DELETE FROM assignment_runs
WHERE id = $1 AND camp_id = $2;
