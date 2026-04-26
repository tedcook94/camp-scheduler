-- name: CreateCamperCabinOverride :one
INSERT INTO camper_cabin_overrides (camp_id, session_id, camper_id, session_age_group_cabin_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, camper_id, session_age_group_cabin_id, created_at;

-- name: DeleteCamperCabinOverride :execrows
DELETE FROM camper_cabin_overrides
WHERE id = $1 AND camp_id = $2;

-- name: GetCamperCabinOverride :one
SELECT id, camp_id, session_id, camper_id, session_age_group_cabin_id, created_at
FROM camper_cabin_overrides
WHERE id = $1 AND camp_id = $2;

-- name: ListCamperCabinOverridesBySession :many
SELECT o.id, o.camp_id, o.session_id, o.camper_id, o.session_age_group_cabin_id, o.created_at,
       sagc.cabin_id AS cabin_id,
       cm.first_name AS camper_first_name,
       cm.last_name AS camper_last_name,
       btrim(cm.first_name || ' ' || cm.last_name)::text AS camper_name,
       cb.cabin_name AS cabin_name,
       ag.age_group_name AS age_group_name
FROM camper_cabin_overrides o
JOIN campers cm ON cm.id = o.camper_id
JOIN session_age_group_cabins sagc ON sagc.id = o.session_age_group_cabin_id
JOIN cabins cb ON cb.id = sagc.cabin_id
JOIN session_age_groups sag ON sag.id = sagc.session_age_group_id
JOIN age_groups ag ON ag.id = sag.age_group_id
WHERE o.session_id = $1 AND o.camp_id = $2
ORDER BY ag.age_group_name, cb.cabin_name, cm.last_name, cm.first_name;

-- name: ListCamperCabinOverridesForSolver :many
SELECT o.camper_id, o.session_age_group_cabin_id
FROM camper_cabin_overrides o
WHERE o.session_id = $1 AND o.camp_id = $2;
