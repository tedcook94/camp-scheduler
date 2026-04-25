-- name: CreateCounselorCabinUnassigned :one
INSERT INTO counselor_cabin_unassigned (camp_id, solution_id, counselor_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, solution_id, counselor_id;

-- name: ListCounselorCabinUnassignedBySolution :many
SELECT u.id, u.camp_id, u.solution_id, u.counselor_id,
       co.first_name AS counselor_first_name,
       co.last_name AS counselor_last_name,
       btrim(co.first_name || ' ' || co.last_name)::text AS counselor_name
FROM counselor_cabin_unassigned u
JOIN counselors co ON co.id = u.counselor_id
WHERE u.solution_id = $1 AND u.camp_id = $2
ORDER BY co.last_name, co.first_name;
