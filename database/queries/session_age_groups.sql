-- name: ListSessionAgeGroups :many
SELECT id, camp_id, session_id, age_group_id, group_size
FROM session_age_groups
WHERE session_id = $1 AND camp_id = $2
ORDER BY age_group_id;

-- name: GetSessionAgeGroup :one
SELECT id, camp_id, session_id, age_group_id, group_size
FROM session_age_groups
WHERE id = $1 AND camp_id = $2 AND session_id = $3;

-- name: CreateSessionAgeGroup :one
INSERT INTO session_age_groups (camp_id, session_id, age_group_id, group_size)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, session_id, age_group_id, group_size;

-- name: UpdateSessionAgeGroup :one
UPDATE session_age_groups
SET age_group_id = $4,
    group_size = $5
WHERE id = $1 AND camp_id = $2 AND session_id = $3
RETURNING id, camp_id, session_id, age_group_id, group_size;

-- name: DeleteSessionAgeGroup :execrows
DELETE FROM session_age_groups
WHERE id = $1 AND camp_id = $2 AND session_id = $3;
