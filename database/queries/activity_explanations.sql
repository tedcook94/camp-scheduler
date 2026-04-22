-- name: CreateActivityExplanation :one
INSERT INTO activity_explanations (camp_id, solution_id, counselor_id, explanation_type, constraint_name, rank, message)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, camp_id, solution_id, counselor_id, explanation_type, constraint_name, rank, message;

-- name: ListActivityExplanationsBySolution :many
SELECT e.id, e.camp_id, e.solution_id, e.counselor_id, e.explanation_type, e.constraint_name, e.rank, e.message,
       co.counselor_name AS counselor_name
FROM activity_explanations e
JOIN counselors co ON co.id = e.counselor_id
WHERE e.solution_id = $1 AND e.camp_id = $2
ORDER BY co.counselor_name, e.explanation_type, COALESCE(e.constraint_name, ''), COALESCE(e.rank, 0), e.message;
