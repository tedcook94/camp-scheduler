-- name: CreateActivitySolution :one
INSERT INTO activity_solutions (camp_id, assignment_run_id, solution_index, score, score_breakdown)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, assignment_run_id, solution_index, score, score_breakdown;

-- name: GetActivitySolution :one
SELECT id, camp_id, assignment_run_id, solution_index, score, score_breakdown
FROM activity_solutions
WHERE id = $1 AND camp_id = $2 AND assignment_run_id = $3;

-- name: ListActivitySolutionsByRun :many
SELECT id, camp_id, assignment_run_id, solution_index, score, score_breakdown
FROM activity_solutions
WHERE assignment_run_id = $1 AND camp_id = $2
ORDER BY solution_index;
