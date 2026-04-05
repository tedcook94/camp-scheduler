-- name: CreateCamperCabinExplanation :one
INSERT INTO camper_cabin_explanations (camp_id, solution_id, camper_id, explanation_type, constraint_name, message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, camp_id, solution_id, camper_id, explanation_type, constraint_name, message;

-- name: ListCamperCabinExplanationsBySolution :many
SELECT id, camp_id, solution_id, camper_id, explanation_type, constraint_name, message
FROM camper_cabin_explanations
WHERE solution_id = $1 AND camp_id = $2
ORDER BY camper_id, explanation_type;
