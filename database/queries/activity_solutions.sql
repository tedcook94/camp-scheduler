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

-- name: GetSelectedActivitySolutionBySession :one
SELECT arss.solution_id
FROM assignment_runs ar
JOIN assignment_run_selected_solutions arss
    ON arss.run_id = ar.id AND arss.camp_id = ar.camp_id
WHERE ar.session_id = @session_id
    AND ar.camp_id = @camp_id
    AND ar.run_type = 'activity_schedule'
ORDER BY ar.created_at DESC
LIMIT 1;
