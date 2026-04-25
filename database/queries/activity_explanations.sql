-- name: CreateActivityExplanation :one
INSERT INTO activity_explanations (camp_id, solution_id, counselor_id, session_activity_id, explanation_type, constraint_name, rank, message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, camp_id, solution_id, counselor_id, session_activity_id, explanation_type, constraint_name, rank, message;

-- name: ListActivityExplanationsBySolution :many
SELECT e.id, e.camp_id, e.solution_id, e.counselor_id, e.session_activity_id, e.explanation_type, e.constraint_name, e.rank, e.message,
       (co.first_name || ' ' || co.last_name)::text AS counselor_name,
       sts.sort_order AS time_slot_sort_order
FROM activity_explanations e
JOIN counselors co ON co.id = e.counselor_id
LEFT JOIN session_activities sa ON sa.id = e.session_activity_id
LEFT JOIN session_time_slots sts ON sts.id = sa.session_time_slot_id
WHERE e.solution_id = $1 AND e.camp_id = $2
ORDER BY co.last_name, co.first_name, e.explanation_type, COALESCE(e.constraint_name, ''), COALESCE(e.rank, 0), COALESCE(sts.sort_order, 0), e.message;
