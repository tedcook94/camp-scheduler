-- name: CreateActivityAssignment :one
INSERT INTO activity_assignments (camp_id, solution_id, counselor_id, session_activity_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, counselor_id, session_activity_id;

-- name: ListActivityAssignmentsBySolution :many
SELECT id, camp_id, solution_id, counselor_id, session_activity_id
FROM activity_assignments
WHERE solution_id = $1 AND camp_id = $2
ORDER BY session_activity_id, counselor_id;
