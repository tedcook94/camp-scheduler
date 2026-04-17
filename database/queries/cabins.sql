-- name: ListCabins :many
SELECT id, camp_id, default_age_group_id, cabin_name
FROM cabins
WHERE camp_id = $1
ORDER BY cabin_name;

-- name: GetCabin :one
SELECT id, camp_id, default_age_group_id, cabin_name
FROM cabins
WHERE id = $1 AND camp_id = $2;

-- name: CreateCabin :one
INSERT INTO cabins (camp_id, default_age_group_id, cabin_name)
VALUES ($1, $2, $3)
RETURNING id, camp_id, default_age_group_id, cabin_name;

-- name: UpdateCabin :one
UPDATE cabins
SET default_age_group_id = $3,
    cabin_name = $4
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, default_age_group_id, cabin_name;

-- name: DeleteCabin :execrows
DELETE FROM cabins
WHERE id = $1 AND camp_id = $2;
