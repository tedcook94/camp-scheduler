-- name: ListCamps :many
SELECT id, camp_name, camp_location, camp_enabled
FROM camps
ORDER BY camp_name;

-- name: GetCamp :one
SELECT id, camp_name, camp_location, camp_enabled
FROM camps
WHERE id = $1;

-- name: CreateCamp :one
INSERT INTO camps (camp_name, camp_location)
VALUES ($1, $2)
RETURNING id, camp_name, camp_location, camp_enabled;

-- name: UpdateCamp :one
UPDATE camps
SET camp_name = @camp_name,
    camp_location = @camp_location,
    camp_enabled = COALESCE(sqlc.narg('camp_enabled'), camp_enabled)
WHERE id = @id
RETURNING id, camp_name, camp_location, camp_enabled;

-- name: DeleteCamp :execrows
DELETE FROM camps
WHERE id = $1;
