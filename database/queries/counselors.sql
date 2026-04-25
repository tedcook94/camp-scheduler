-- name: ListCounselors :many
SELECT id, camp_id, first_name, last_name, junior_counselor, counselor_enabled, gender
FROM counselors
WHERE camp_id = $1
ORDER BY last_name, first_name;

-- name: GetCounselor :one
SELECT id, camp_id, first_name, last_name, junior_counselor, counselor_enabled, gender
FROM counselors
WHERE id = $1 AND camp_id = $2;

-- name: CreateCounselor :one
INSERT INTO counselors (camp_id, first_name, last_name, junior_counselor, gender)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, first_name, last_name, junior_counselor, counselor_enabled, gender;

-- name: UpdateCounselor :one
UPDATE counselors
SET first_name = $3,
    last_name = $4,
    junior_counselor = $5,
    counselor_enabled = $6,
    gender = $7
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, first_name, last_name, junior_counselor, counselor_enabled, gender;

-- name: DeleteCounselor :execrows
DELETE FROM counselors
WHERE id = $1 AND camp_id = $2;

-- name: ListEnabledCounselors :many
SELECT id, camp_id, first_name, last_name, junior_counselor, counselor_enabled, gender
FROM counselors
WHERE camp_id = $1 AND counselor_enabled = true
ORDER BY last_name, first_name;
