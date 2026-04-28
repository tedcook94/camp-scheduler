-- name: CreateCounselorCabinOverride :one
INSERT INTO counselor_cabin_overrides (camp_id, session_id, counselor_id, session_cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, counselor_id, session_cabin_id, created_at;

-- name: DeleteCounselorCabinOverride :execrows
DELETE FROM counselor_cabin_overrides
WHERE id = $1 AND camp_id = $2;

-- name: GetCounselorCabinOverride :one
SELECT id, camp_id, session_id, counselor_id, session_cabin_id, created_at
FROM counselor_cabin_overrides
WHERE id = $1 AND camp_id = $2;

-- name: ListCounselorCabinOverridesBySession :many
SELECT o.id, o.camp_id, o.session_id, o.counselor_id, o.session_cabin_id, o.created_at,
       sagc.cabin_id AS cabin_id,
       co.first_name AS counselor_first_name,
       co.last_name AS counselor_last_name,
       btrim(co.first_name || ' ' || co.last_name)::text AS counselor_name,
       cb.cabin_name AS cabin_name,
       ag.age_group_name AS age_group_name
FROM counselor_cabin_overrides o
JOIN counselors co ON co.id = o.counselor_id
JOIN session_cabins sagc ON sagc.id = o.session_cabin_id
JOIN cabins cb ON cb.id = sagc.cabin_id
JOIN session_age_groups sag ON sag.id = sagc.session_age_group_id
JOIN age_groups ag ON ag.id = sag.age_group_id
WHERE o.session_id = $1 AND o.camp_id = $2
ORDER BY ag.age_group_name, cb.cabin_name, co.last_name, co.first_name;

-- name: ListCounselorCabinOverridesForSolver :many
SELECT o.counselor_id, o.session_cabin_id
FROM counselor_cabin_overrides o
WHERE o.session_id = $1 AND o.camp_id = $2;
