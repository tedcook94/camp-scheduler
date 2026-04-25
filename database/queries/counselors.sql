-- name: ListCounselors :many
SELECT id, camp_id, first_name, last_name, junior_counselor, archived, gender
FROM counselors
WHERE camp_id = $1 AND archived = false
ORDER BY last_name, first_name;

-- name: ListAllCounselors :many
SELECT id, camp_id, first_name, last_name, junior_counselor, archived, gender
FROM counselors
WHERE camp_id = $1
ORDER BY last_name, first_name;

-- name: ListArchivedCounselors :many
SELECT id, camp_id, first_name, last_name, junior_counselor, archived, gender
FROM counselors
WHERE camp_id = $1 AND archived = true
ORDER BY last_name, first_name;

-- name: GetCounselor :one
SELECT id, camp_id, first_name, last_name, junior_counselor, archived, gender
FROM counselors
WHERE id = $1 AND camp_id = $2;

-- name: CreateCounselor :one
INSERT INTO counselors (camp_id, first_name, last_name, junior_counselor, gender)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, first_name, last_name, junior_counselor, archived, gender;

-- name: UpdateCounselor :one
UPDATE counselors
SET first_name = $3,
    last_name = $4,
    junior_counselor = $5,
    gender = $6
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, first_name, last_name, junior_counselor, archived, gender;

-- name: DeleteCounselor :execrows
DELETE FROM counselors
WHERE id = $1 AND camp_id = $2;

-- name: ArchiveCounselor :execrows
UPDATE counselors
SET archived = true
WHERE id = $1 AND camp_id = $2;

-- name: UnarchiveCounselor :execrows
UPDATE counselors
SET archived = false
WHERE id = $1 AND camp_id = $2;
