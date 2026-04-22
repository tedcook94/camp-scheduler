-- name: CreateCamperCabinExplanation :one
INSERT INTO camper_cabin_explanations (camp_id, solution_id, camper_id, explanation_type, constraint_name, message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, camp_id, solution_id, camper_id, explanation_type, constraint_name, message;

-- name: ListCamperCabinExplanationsBySolution :many
SELECT e.id, e.camp_id, e.solution_id, e.camper_id, e.explanation_type, e.constraint_name, e.message,
       cm.camper_name AS camper_name
FROM camper_cabin_explanations e
JOIN campers cm ON cm.id = e.camper_id
WHERE e.solution_id = $1 AND e.camp_id = $2
ORDER BY cm.camper_name, e.explanation_type, e.message;
