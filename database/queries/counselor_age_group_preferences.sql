-- name: ListCounselorAgeGroupPreferences :many
SELECT id, camp_id, counselor_id, session_id, age_group_id, rank
FROM counselor_age_group_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: CreateCounselorAgeGroupPreference :one
INSERT INTO counselor_age_group_preferences (camp_id, counselor_id, session_id, age_group_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, age_group_id, rank;

-- name: DeleteAllCounselorAgeGroupPreferences :exec
DELETE FROM counselor_age_group_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3;

-- name: ListSessionAgeGroupPreferences :many
SELECT counselor_id, age_group_id, rank
FROM counselor_age_group_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY counselor_id, rank;
