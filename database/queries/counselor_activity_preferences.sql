-- name: ListCounselorActivityPreferences :many
SELECT id, camp_id, counselor_id, session_id, activity_id, rank
FROM counselor_activity_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: CreateCounselorActivityPreference :one
INSERT INTO counselor_activity_preferences (camp_id, counselor_id, session_id, activity_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, activity_id, rank;

-- name: DeleteAllCounselorActivityPreferences :exec
DELETE FROM counselor_activity_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3;

-- name: ListSessionActivityPreferences :many
SELECT counselor_id, activity_id, rank
FROM counselor_activity_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY counselor_id, rank;
