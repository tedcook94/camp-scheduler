-- name: ListSessionCabins :many
SELECT
    c.id,
    c.cabin_name,
    sagc.gender,
    sag.age_group_id,
    ag.age_group_name,
    sagc.id AS session_age_group_cabin_id,
    sagc.required_counselors
FROM session_age_groups sag
JOIN session_age_group_cabins sagc ON sagc.session_age_group_id = sag.id
JOIN cabins c ON c.id = sagc.cabin_id
JOIN age_groups ag ON ag.id = sag.age_group_id
WHERE sag.session_id = $1 AND sag.camp_id = $2
ORDER BY c.cabin_name;

-- name: ListSessionAgeGroupCabins :many
SELECT id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors, gender
FROM session_age_group_cabins
WHERE session_age_group_id = $1 AND camp_id = $2
ORDER BY cabin_id;

-- name: ListSessionAgeGroupCabinsBySession :many
SELECT sagc.id, sagc.camp_id, sagc.session_age_group_id, sagc.cabin_id, sagc.group_size, sagc.required_counselors, sagc.gender
FROM session_age_group_cabins sagc
JOIN session_age_groups sag ON sag.id = sagc.session_age_group_id
WHERE sag.session_id = $1 AND sagc.camp_id = $2
ORDER BY sagc.cabin_id;

-- name: GetSessionAgeGroupCabin :one
SELECT sagc.id, sagc.camp_id, sagc.session_age_group_id, sagc.cabin_id, sagc.group_size, sagc.required_counselors, sagc.gender
FROM session_age_group_cabins sagc
JOIN session_age_groups sag ON sag.id = sagc.session_age_group_id
WHERE sagc.id = $1 AND sagc.camp_id = $2 AND sag.session_id = $3;

-- name: CreateSessionAgeGroupCabin :one
INSERT INTO session_age_group_cabins (camp_id, session_age_group_id, cabin_id, group_size, required_counselors, gender)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, camp_id, session_age_group_id, cabin_id, group_size, required_counselors, gender;

-- name: UpdateSessionAgeGroupCabin :one
UPDATE session_age_group_cabins sagc
SET cabin_id = $4,
    group_size = $5,
    required_counselors = $6,
    gender = $7
FROM session_age_groups sag
WHERE sagc.id = $1 AND sagc.camp_id = $2
  AND sag.id = sagc.session_age_group_id AND sag.session_id = $3
RETURNING sagc.id, sagc.camp_id, sagc.session_age_group_id, sagc.cabin_id, sagc.group_size, sagc.required_counselors, sagc.gender;

-- name: DeleteSessionAgeGroupCabin :execrows
DELETE FROM session_age_group_cabins sagc
USING session_age_groups sag
WHERE sagc.id = $1 AND sagc.camp_id = $2
  AND sag.id = sagc.session_age_group_id AND sag.session_id = $3;

-- name: ListSessionCabinsWithCapacity :many
SELECT
    c.id,
    c.cabin_name,
    sagc.gender,
    c.archived AS cabin_archived,
    sag.age_group_id,
    ag.age_group_name,
    ag.archived AS age_group_archived,
    sagc.id AS session_age_group_cabin_id,
    sagc.group_size,
    sagc.required_counselors
FROM session_age_groups sag
JOIN session_age_group_cabins sagc ON sagc.session_age_group_id = sag.id
JOIN cabins c ON c.id = sagc.cabin_id
JOIN age_groups ag ON ag.id = sag.age_group_id AND ag.camp_id = sag.camp_id
WHERE sag.session_id = $1 AND sag.camp_id = $2
ORDER BY c.cabin_name;
