-- name: ListCampers :many
SELECT id, camp_id, first_name, last_name, gender, archived
FROM campers
WHERE camp_id = $1
ORDER BY last_name, first_name;

-- name: GetCamper :one
SELECT id, camp_id, first_name, last_name, gender, archived
FROM campers
WHERE id = $1 AND camp_id = $2;

-- name: CreateCamper :one
INSERT INTO campers (camp_id, first_name, last_name, gender)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, first_name, last_name, gender, archived;

-- name: UpdateCamper :one
UPDATE campers
SET first_name = $3,
    last_name = $4,
    gender = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, first_name, last_name, gender, archived;

-- name: DeleteCamper :execrows
DELETE FROM campers
WHERE id = $1 AND camp_id = $2;
