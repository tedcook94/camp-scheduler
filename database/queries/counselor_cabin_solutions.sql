-- name: CreateCounselorCabinSolution :one
INSERT INTO counselor_cabin_solutions (camp_id, assignment_run_id, solution_index, score, score_breakdown)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, assignment_run_id, solution_index, score, score_breakdown;

-- name: GetCounselorCabinSolution :one
SELECT id, camp_id, assignment_run_id, solution_index, score, score_breakdown
FROM counselor_cabin_solutions
WHERE id = $1 AND camp_id = $2 AND assignment_run_id = $3;

-- name: ListCounselorCabinSolutionsByRun :many
SELECT id, camp_id, assignment_run_id, solution_index, score, score_breakdown
FROM counselor_cabin_solutions
WHERE assignment_run_id = $1 AND camp_id = $2
ORDER BY solution_index;
