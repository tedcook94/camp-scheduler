-- name: ListCampers :many
SELECT id, camp_id, camper_name
FROM campers
WHERE camp_id = $1
ORDER BY camper_name;

-- name: GetCamper :one
SELECT id, camp_id, camper_name
FROM campers
WHERE id = $1 AND camp_id = $2;

-- name: CreateCamper :one
INSERT INTO campers (camp_id, camper_name)
VALUES ($1, $2)
RETURNING id, camp_id, camper_name;

-- name: UpdateCamper :one
UPDATE campers
SET camper_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, camper_name;

-- name: DeleteCamper :execrows
DELETE FROM campers
WHERE id = $1 AND camp_id = $2;
