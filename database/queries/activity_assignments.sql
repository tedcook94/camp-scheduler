-- name: CreateActivityAssignment :one
INSERT INTO activity_assignments (camp_id, solution_id, counselor_id, session_activity_id)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, solution_id, counselor_id, session_activity_id;

-- name: ListActivityAssignmentsBySolution :many
SELECT a.id, a.camp_id, a.solution_id, a.counselor_id, a.session_activity_id,
       (co.first_name || ' ' || co.last_name)::text AS counselor_name,
       act.activity_name AS activity_name,
       ts.time_slot_name AS time_slot_name,
       sts.sort_order AS sort_order
FROM activity_assignments a
JOIN counselors co ON co.id = a.counselor_id
JOIN session_activities sa ON sa.id = a.session_activity_id
JOIN activities act ON act.id = sa.activity_id
JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
JOIN time_slots ts ON ts.id = sts.time_slot_id
WHERE a.solution_id = $1 AND a.camp_id = $2
ORDER BY sts.sort_order, sts.id, act.activity_name, co.last_name, co.first_name;
