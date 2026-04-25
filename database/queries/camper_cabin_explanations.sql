-- name: CreateCamperCabinExplanation :one
INSERT INTO camper_cabin_explanations (camp_id, solution_id, camper_id, explanation_type, constraint_name, rank, message)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, camp_id, solution_id, camper_id, explanation_type, constraint_name, rank, message;

-- name: ListCamperCabinExplanationsBySolution :many
SELECT e.id, e.camp_id, e.solution_id, e.camper_id, e.explanation_type, e.constraint_name, e.rank, e.message,
       cm.first_name AS camper_first_name,
       cm.last_name AS camper_last_name,
       btrim(cm.first_name || ' ' || cm.last_name)::text AS camper_name
FROM camper_cabin_explanations e
JOIN campers cm ON cm.id = e.camper_id
WHERE e.solution_id = $1 AND e.camp_id = $2
ORDER BY cm.last_name, cm.first_name, e.explanation_type, COALESCE(e.constraint_name, ''), COALESCE(e.rank, 0), e.message;
