-- name: CreateActivityExplanation :one
INSERT INTO activity_explanations (camp_id, solution_id, counselor_id, explanation_type, constraint_name, message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, camp_id, solution_id, counselor_id, explanation_type, constraint_name, message;

-- name: ListActivityExplanationsBySolution :many
SELECT id, camp_id, solution_id, counselor_id, explanation_type, constraint_name, message
FROM activity_explanations
WHERE solution_id = $1 AND camp_id = $2
ORDER BY counselor_id, explanation_type, message;
