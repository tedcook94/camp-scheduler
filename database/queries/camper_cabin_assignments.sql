-- name: CreateCamperCabinAssignment :one
INSERT INTO camper_cabin_assignments (camp_id, solution_id, camper_id, cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, camper_id, cabin_id;

-- name: ListCamperCabinAssignmentsBySolution :many
SELECT a.id, a.camp_id, a.solution_id, a.camper_id, a.cabin_id,
       cm.camper_name AS camper_name,
       cb.cabin_name AS cabin_name,
       ag.age_group_name AS age_group_name
FROM camper_cabin_assignments a
JOIN campers cm ON cm.id = a.camper_id
JOIN cabins cb ON cb.id = a.cabin_id
JOIN age_groups ag ON ag.id = cb.default_age_group_id
WHERE a.solution_id = $1 AND a.camp_id = $2
ORDER BY ag.age_group_name, cb.cabin_name, cm.camper_name;
