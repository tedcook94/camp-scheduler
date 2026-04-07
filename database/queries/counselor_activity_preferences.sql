-- name: ListCounselorActivityPreferences :many
SELECT id, camp_id, counselor_id, session_id, activity_id, rank
FROM counselor_activity_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: GetCounselorActivityPreference :one
SELECT id, camp_id, counselor_id, session_id, activity_id, rank
FROM counselor_activity_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4;

-- name: CreateCounselorActivityPreference :one
INSERT INTO counselor_activity_preferences (camp_id, counselor_id, session_id, activity_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, activity_id, rank;

-- name: UpdateCounselorActivityPreference :one
UPDATE counselor_activity_preferences
SET activity_id = $5,
    rank = $6
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4
RETURNING id, camp_id, counselor_id, session_id, activity_id, rank;

-- name: DeleteCounselorActivityPreference :execrows
DELETE FROM counselor_activity_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4;

-- name: ListSessionActivityPreferences :many
SELECT counselor_id, activity_id, rank
FROM counselor_activity_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY counselor_id, rank;
