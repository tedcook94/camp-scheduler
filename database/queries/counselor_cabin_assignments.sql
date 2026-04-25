-- name: CreateCounselorCabinAssignment :one
INSERT INTO counselor_cabin_assignments (camp_id, solution_id, counselor_id, session_age_group_cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, counselor_id, session_age_group_cabin_id;

-- name: ListCounselorCabinAssignmentsBySolution :many
SELECT a.id, a.camp_id, a.solution_id, a.counselor_id,
       sagc.cabin_id AS cabin_id,
       co.first_name AS counselor_first_name,
       co.last_name AS counselor_last_name,
       btrim(co.first_name || ' ' || co.last_name)::text AS counselor_name,
       cb.cabin_name AS cabin_name,
       ag.age_group_name AS age_group_name
FROM counselor_cabin_assignments a
JOIN counselors co ON co.id = a.counselor_id
JOIN session_age_group_cabins sagc ON sagc.id = a.session_age_group_cabin_id
JOIN cabins cb ON cb.id = sagc.cabin_id
JOIN session_age_groups sag ON sag.id = sagc.session_age_group_id
JOIN age_groups ag ON ag.id = sag.age_group_id
WHERE a.solution_id = $1 AND a.camp_id = $2
ORDER BY ag.age_group_name, cb.cabin_name, co.last_name, co.first_name;
