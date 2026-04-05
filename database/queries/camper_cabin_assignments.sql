-- name: CreateCamperCabinAssignment :one
INSERT INTO camper_cabin_assignments (camp_id, solution_id, camper_id, cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, camper_id, cabin_id;

-- name: ListCamperCabinAssignmentsBySolution :many
SELECT id, camp_id, solution_id, camper_id, cabin_id
FROM camper_cabin_assignments
WHERE solution_id = $1 AND camp_id = $2
ORDER BY cabin_id, camper_id;
