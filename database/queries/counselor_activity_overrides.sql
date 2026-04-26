-- name: CreateCounselorActivityOverride :one
INSERT INTO counselor_activity_overrides (camp_id, session_id, counselor_id, session_activity_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, counselor_id, session_activity_id, created_at;

-- name: DeleteCounselorActivityOverride :execrows
DELETE FROM counselor_activity_overrides
WHERE id = $1 AND camp_id = $2;

-- name: GetCounselorActivityOverride :one
SELECT id, camp_id, session_id, counselor_id, session_activity_id, created_at
FROM counselor_activity_overrides
WHERE id = $1 AND camp_id = $2;

-- name: ListCounselorActivityOverridesBySession :many
SELECT o.id, o.camp_id, o.session_id, o.counselor_id, o.session_activity_id, o.created_at,
       co.first_name AS counselor_first_name,
       co.last_name AS counselor_last_name,
       btrim(co.first_name || ' ' || co.last_name)::text AS counselor_name,
       sa.session_time_slot_id AS session_time_slot_id,
       act.activity_name AS activity_name,
       ts.time_slot_name AS time_slot_name,
       sts.sort_order AS sort_order
FROM counselor_activity_overrides o
JOIN counselors co ON co.id = o.counselor_id
JOIN session_activities sa ON sa.id = o.session_activity_id
JOIN activities act ON act.id = sa.activity_id
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
JOIN time_slots ts ON ts.id = sts.time_slot_id
WHERE o.session_id = $1 AND o.camp_id = $2
ORDER BY sts.sort_order, sts.id, act.activity_name, co.last_name, co.first_name;

-- name: ListCounselorActivityOverridesForSolver :many
SELECT o.counselor_id, o.session_activity_id, sa.session_time_slot_id
FROM counselor_activity_overrides o
JOIN session_activities sa ON sa.id = o.session_activity_id
WHERE o.session_id = $1 AND o.camp_id = $2;
