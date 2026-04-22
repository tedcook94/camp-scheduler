-- name: CreateCounselorCabinAssignment :one
INSERT INTO counselor_cabin_assignments (camp_id, solution_id, counselor_id, cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, counselor_id, cabin_id;

-- name: ListCounselorCabinAssignmentsBySolution :many
SELECT a.id, a.camp_id, a.solution_id, a.counselor_id, a.cabin_id,
       co.counselor_name AS counselor_name,
       cb.cabin_name AS cabin_name,
       ag.age_group_name AS age_group_name
FROM counselor_cabin_assignments a
JOIN counselors co ON co.id = a.counselor_id
JOIN cabins cb ON cb.id = a.cabin_id
JOIN age_groups ag ON ag.id = cb.default_age_group_id
WHERE a.solution_id = $1 AND a.camp_id = $2
ORDER BY ag.age_group_name, cb.cabin_name, co.counselor_name;
