-- name: ListAgeGroups :many
SELECT id, camp_id, age_group_name, archived
FROM age_groups
WHERE camp_id = $1 AND archived = false
ORDER BY age_group_name;

-- name: ListAllAgeGroups :many
SELECT id, camp_id, age_group_name, archived
FROM age_groups
WHERE camp_id = $1
ORDER BY age_group_name;

-- name: ListArchivedAgeGroups :many
SELECT id, camp_id, age_group_name, archived
FROM age_groups
WHERE camp_id = $1 AND archived = true
ORDER BY age_group_name;

-- name: GetAgeGroup :one
SELECT id, camp_id, age_group_name, archived
FROM age_groups
WHERE id = $1 AND camp_id = $2;

-- name: CreateAgeGroup :one
INSERT INTO age_groups (camp_id, age_group_name)
VALUES ($1, $2)
RETURNING id, camp_id, age_group_name, archived;

-- name: UpdateAgeGroup :one
UPDATE age_groups
SET age_group_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, age_group_name, archived;

-- name: DeleteAgeGroup :execrows
DELETE FROM age_groups
WHERE id = $1 AND camp_id = $2;

-- name: ArchiveAgeGroup :execrows
UPDATE age_groups
SET archived = true
WHERE id = $1 AND camp_id = $2;

-- name: UnarchiveAgeGroup :execrows
UPDATE age_groups
SET archived = false
WHERE id = $1 AND camp_id = $2;
