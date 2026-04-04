-- name: ListCounselorSessionHistory :many
SELECT id, camp_id, counselor_id, session_id, age_group_id, cabin_id
FROM counselor_session_history
WHERE counselor_id = $1 AND camp_id = $2
ORDER BY session_id;

-- name: GetCounselorSessionHistoryEntry :one
SELECT id, camp_id, counselor_id, session_id, age_group_id, cabin_id
FROM counselor_session_history
WHERE id = $1 AND camp_id = $2;

-- name: CreateCounselorSessionHistoryEntry :one
INSERT INTO counselor_session_history (camp_id, counselor_id, session_id, age_group_id, cabin_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, age_group_id, cabin_id;

-- name: UpdateCounselorSessionHistoryEntry :one
UPDATE counselor_session_history
SET session_id = $3,
    age_group_id = $4,
    cabin_id = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, counselor_id, session_id, age_group_id, cabin_id;

-- name: DeleteCounselorSessionHistoryEntry :execrows
DELETE FROM counselor_session_history
WHERE id = $1 AND camp_id = $2;

-- name: GetCounselorHistorySummary :many
SELECT
    csh.id,
    s.session_name,
    se.id AS season_id,
    se.season_name,
    ag.age_group_name,
    c.cabin_name
FROM counselor_session_history csh
JOIN sessions s ON s.id = csh.session_id
JOIN seasons se ON se.id = s.season_id
JOIN age_groups ag ON ag.id = csh.age_group_id
LEFT JOIN cabins c ON c.id = csh.cabin_id
WHERE csh.counselor_id = $1
  AND csh.camp_id = $2
  AND (sqlc.narg('season_id')::uuid IS NULL OR se.id = sqlc.narg('season_id')::uuid)
ORDER BY se.season_name, s.session_name;
