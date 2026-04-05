-- name: ListCounselorAgeGroupPreferences :many
SELECT id, camp_id, counselor_id, session_id, age_group_id, rank
FROM counselor_age_group_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: GetCounselorAgeGroupPreference :one
SELECT id, camp_id, counselor_id, session_id, age_group_id, rank
FROM counselor_age_group_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4;

-- name: CreateCounselorAgeGroupPreference :one
INSERT INTO counselor_age_group_preferences (camp_id, counselor_id, session_id, age_group_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, age_group_id, rank;

-- name: UpdateCounselorAgeGroupPreference :one
UPDATE counselor_age_group_preferences
SET age_group_id = $5,
    rank = $6
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4
RETURNING id, camp_id, counselor_id, session_id, age_group_id, rank;

-- name: DeleteCounselorAgeGroupPreference :execrows
DELETE FROM counselor_age_group_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND counselor_id = $4;

-- name: ListSessionAgeGroupPreferences :many
SELECT counselor_id, age_group_id, rank
FROM counselor_age_group_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY counselor_id, rank;
