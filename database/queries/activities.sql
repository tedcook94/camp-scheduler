-- name: ListActivities :many
SELECT id, camp_id, activity_name, archived
FROM activities
WHERE camp_id = $1
ORDER BY activity_name;

-- name: GetActivity :one
SELECT id, camp_id, activity_name, archived
FROM activities
WHERE id = $1 AND camp_id = $2;

-- name: CreateActivity :one
INSERT INTO activities (camp_id, activity_name)
VALUES ($1, $2)
RETURNING id, camp_id, activity_name, archived;

-- name: UpdateActivity :one
UPDATE activities
SET activity_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, activity_name, archived;

-- name: DeleteActivity :execrows
DELETE FROM activities
WHERE id = $1 AND camp_id = $2;
