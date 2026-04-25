-- name: CreateAssignmentRun :one
INSERT INTO assignment_runs (camp_id, session_id, run_type, status)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, run_type, status, created_at, is_stale;

-- name: GetAssignmentRun :one
SELECT id, camp_id, session_id, run_type, status, created_at, is_stale
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
    ar.is_stale,
    arss.solution_id AS selected_solution_id
FROM assignment_runs ar
LEFT JOIN assignment_run_selected_solutions arss ON arss.run_id = ar.id AND arss.camp_id = ar.camp_id
WHERE ar.session_id = $1 AND ar.camp_id = $2
ORDER BY ar.created_at DESC;

-- name: UpdateAssignmentRunStatus :one
UPDATE assignment_runs
SET status = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, session_id, run_type, status, created_at, is_stale;

-- name: DeleteAssignmentRun :execrows
DELETE FROM assignment_runs
WHERE id = $1 AND camp_id = $2;

-- name: DeleteAssignmentRunsBySessionAndType :execrows
DELETE FROM assignment_runs
WHERE camp_id = $1 AND session_id = $2 AND run_type = $3;

-- name: LockAssignmentRunsBySessionAndType :many
SELECT id
FROM assignment_runs
WHERE camp_id = $1 AND session_id = $2 AND run_type = $3
ORDER BY id
FOR UPDATE;

-- name: MarkAssignmentRunsStale :execrows
UPDATE assignment_runs
SET is_stale = true
WHERE camp_id = $1
  AND session_id = ANY(@session_ids::uuid[])
  AND run_type = ANY(@run_types::text[])
  AND is_stale = false;

-- name: ListSessionsByPreviousSession :many
SELECT id
FROM sessions
WHERE camp_id = $1
  AND previous_session = ANY(@previous_session_ids::uuid[]);

-- name: GetSelectedRunBySessionAndType :one
SELECT id
FROM assignment_runs
WHERE camp_id = $1 AND session_id = $2 AND run_type = $3 AND status = 'selected';
