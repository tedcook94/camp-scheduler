-- name: ListCabins :many
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.camp_id = $1
ORDER BY c.cabin_name;

-- name: GetCabin :one
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.id = $1 AND c.camp_id = $2;

-- name: CreateCabin :one
WITH inserted AS (
    INSERT INTO cabins (camp_id, default_age_group_id, cabin_name)
    VALUES ($1, $2, $3)
    RETURNING id, camp_id, default_age_group_id, cabin_name
)
SELECT i.id, i.camp_id, i.default_age_group_id, i.cabin_name,
       ag.age_group_name AS default_age_group_name
FROM inserted i
JOIN age_groups ag ON ag.id = i.default_age_group_id;

-- name: UpdateCabin :one
WITH updated AS (
    UPDATE cabins
    SET default_age_group_id = $3,
        cabin_name = $4
    WHERE cabins.id = $1 AND cabins.camp_id = $2
    RETURNING cabins.id, cabins.camp_id, cabins.default_age_group_id, cabins.cabin_name
)
SELECT u.id, u.camp_id, u.default_age_group_id, u.cabin_name,
       ag.age_group_name AS default_age_group_name
FROM updated u
JOIN age_groups ag ON ag.id = u.default_age_group_id;

-- name: DeleteCabin :execrows
DELETE FROM cabins
WHERE id = $1 AND camp_id = $2;
