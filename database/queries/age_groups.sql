-- name: ListAgeGroups :many
SELECT id, camp_id, age_group_name
FROM age_groups
WHERE camp_id = $1
ORDER BY age_group_name;

-- name: GetAgeGroup :one
SELECT id, camp_id, age_group_name
FROM age_groups
WHERE id = $1 AND camp_id = $2;

-- name: CreateAgeGroup :one
INSERT INTO age_groups (camp_id, age_group_name)
VALUES ($1, $2)
RETURNING id, camp_id, age_group_name;

-- name: UpdateAgeGroup :one
UPDATE age_groups
SET age_group_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, age_group_name;

-- name: DeleteAgeGroup :execrows
DELETE FROM age_groups
WHERE id = $1 AND camp_id = $2;
