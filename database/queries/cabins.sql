-- name: ListCabins :many
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       c.default_group_size, c.default_required_counselors, c.gender, c.archived,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.camp_id = $1 AND c.archived = false
ORDER BY c.cabin_name;

-- name: ListAllCabins :many
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       c.default_group_size, c.default_required_counselors, c.gender, c.archived,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.camp_id = $1
ORDER BY c.cabin_name;

-- name: ListArchivedCabins :many
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       c.default_group_size, c.default_required_counselors, c.gender, c.archived,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.camp_id = $1 AND c.archived = true
ORDER BY c.cabin_name;

-- name: GetCabin :one
SELECT c.id, c.camp_id, c.default_age_group_id, c.cabin_name,
       c.default_group_size, c.default_required_counselors, c.gender, c.archived,
       ag.age_group_name AS default_age_group_name
FROM cabins c
JOIN age_groups ag ON ag.id = c.default_age_group_id
WHERE c.id = $1 AND c.camp_id = $2;

-- name: CreateCabin :one
WITH inserted AS (
    INSERT INTO cabins (camp_id, default_age_group_id, cabin_name,
                        default_group_size, default_required_counselors, gender)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, camp_id, default_age_group_id, cabin_name,
              default_group_size, default_required_counselors, gender, archived
)
SELECT i.id, i.camp_id, i.default_age_group_id, i.cabin_name,
       i.default_group_size, i.default_required_counselors, i.gender, i.archived,
       ag.age_group_name AS default_age_group_name
FROM inserted i
JOIN age_groups ag ON ag.id = i.default_age_group_id;

-- name: UpdateCabin :one
WITH updated AS (
    UPDATE cabins
    SET default_age_group_id = $3,
        cabin_name = $4,
        default_group_size = $5,
        default_required_counselors = $6,
        gender = $7
    WHERE cabins.id = $1 AND cabins.camp_id = $2
    RETURNING cabins.id, cabins.camp_id, cabins.default_age_group_id, cabins.cabin_name,
              cabins.default_group_size, cabins.default_required_counselors, cabins.gender, cabins.archived
)
SELECT u.id, u.camp_id, u.default_age_group_id, u.cabin_name,
       u.default_group_size, u.default_required_counselors, u.gender, u.archived,
       ag.age_group_name AS default_age_group_name
FROM updated u
JOIN age_groups ag ON ag.id = u.default_age_group_id;

-- name: DeleteCabin :execrows
DELETE FROM cabins
WHERE id = $1 AND camp_id = $2;
