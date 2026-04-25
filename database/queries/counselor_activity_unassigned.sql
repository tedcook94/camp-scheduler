-- name: CreateCounselorActivityUnassigned :one
INSERT INTO counselor_activity_unassigned (camp_id, solution_id, counselor_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, solution_id, counselor_id;

-- name: CreateCounselorActivityUnassignedSlot :one
INSERT INTO counselor_activity_unassigned_slots (camp_id, unassigned_id, session_time_slot_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, unassigned_id, session_time_slot_id;

-- name: ListCounselorActivityUnassignedBySolution :many
SELECT u.id, u.camp_id, u.solution_id, u.counselor_id,
       (co.first_name || ' ' || co.last_name)::text AS counselor_name
FROM counselor_activity_unassigned u
JOIN counselors co ON co.id = u.counselor_id
WHERE u.solution_id = $1 AND u.camp_id = $2
ORDER BY co.last_name, co.first_name;

-- name: ListCounselorActivityUnassignedSlotsBySolution :many
SELECT s.id, s.camp_id, s.unassigned_id, s.session_time_slot_id,
       u.counselor_id,
       ts.time_slot_name AS time_slot_name,
       sts.sort_order AS sort_order
FROM counselor_activity_unassigned_slots s
JOIN counselor_activity_unassigned u ON u.id = s.unassigned_id
JOIN session_time_slots sts ON sts.id = s.session_time_slot_id
JOIN time_slots ts ON ts.id = sts.time_slot_id
WHERE u.solution_id = $1 AND s.camp_id = $2
ORDER BY u.counselor_id, sts.sort_order, ts.time_slot_name;
