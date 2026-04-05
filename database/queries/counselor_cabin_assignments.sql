-- name: CreateCounselorCabinAssignment :one
INSERT INTO counselor_cabin_assignments (camp_id, solution_id, counselor_id, cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, counselor_id, cabin_id;

-- name: ListCounselorCabinAssignmentsBySolution :many
SELECT id, camp_id, solution_id, counselor_id, cabin_id
FROM counselor_cabin_assignments
WHERE solution_id = $1 AND camp_id = $2;
