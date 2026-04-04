-- name: ListCounselors :many
SELECT id, camp_id, counselor_name, junior_counselor, counselor_enabled
FROM counselors
WHERE camp_id = $1
ORDER BY counselor_name;

-- name: GetCounselor :one
SELECT id, camp_id, counselor_name, junior_counselor, counselor_enabled
FROM counselors
WHERE id = $1 AND camp_id = $2;

-- name: CreateCounselor :one
INSERT INTO counselors (camp_id, counselor_name, junior_counselor)
VALUES ($1, $2, $3)
RETURNING id, camp_id, counselor_name, junior_counselor, counselor_enabled;

-- name: UpdateCounselor :one
UPDATE counselors
SET counselor_name = $3,
    junior_counselor = $4,
    counselor_enabled = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, counselor_name, junior_counselor, counselor_enabled;

-- name: DeleteCounselor :execrows
DELETE FROM counselors
WHERE id = $1 AND camp_id = $2;
