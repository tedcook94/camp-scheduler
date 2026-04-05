-- name: ListSessionCabins :many
SELECT
    c.id,
    c.cabin_name,
    c.age_group_id,
    sagc.required_counselors
FROM session_age_groups sag
JOIN session_age_group_cabins sagc ON sagc.session_age_group_id = sag.id
JOIN cabins c ON c.id = sagc.cabin_id
WHERE sag.session_id = $1 AND sag.camp_id = $2
ORDER BY c.cabin_name;

-- name: ListSessionAgeGroupCabins :many
SELECT id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors
FROM session_age_group_cabins
WHERE session_age_group_id = $1 AND camp_id = $2
ORDER BY cabin_id;

-- name: GetSessionAgeGroupCabin :one
SELECT id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors
FROM session_age_group_cabins
WHERE id = $1 AND camp_id = $2;

-- name: CreateSessionAgeGroupCabin :one
INSERT INTO session_age_group_cabins (camp_id, session_age_group_id, cabin_id, group_size, required_counselors)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors;

-- name: UpdateSessionAgeGroupCabin :one
UPDATE session_age_group_cabins
SET cabin_id = $3,
    group_size = $4,
    required_counselors = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors;

-- name: DeleteSessionAgeGroupCabin :execrows
DELETE FROM session_age_group_cabins
WHERE id = $1 AND camp_id = $2;
