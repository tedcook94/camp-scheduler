-- name: CreateCounselorCabinExplanation :one
INSERT INTO counselor_cabin_explanations (camp_id, solution_id, counselor_id, explanation_type, constraint_name, rank, message)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, camp_id, solution_id, counselor_id, explanation_type, constraint_name, rank, message;

-- name: ListCounselorCabinExplanationsBySolution :many
SELECT e.id, e.camp_id, e.solution_id, e.counselor_id, e.explanation_type, e.constraint_name, e.rank, e.message,
       co.first_name AS counselor_first_name,
       co.last_name AS counselor_last_name,
       btrim(co.first_name || ' ' || co.last_name)::text AS counselor_name
FROM counselor_cabin_explanations e
JOIN counselors co ON co.id = e.counselor_id
WHERE e.solution_id = $1 AND e.camp_id = $2
ORDER BY co.last_name, co.first_name, e.explanation_type, COALESCE(e.constraint_name, ''), COALESCE(e.rank, 0), e.message;
