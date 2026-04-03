-- name: ListCounselors :many
SELECT id, camp_id, counselor_name, junior_counselor, counselor_enabled
FROM counselors
WHERE camp_id = $1
ORDER BY counselor_name;

-- name: GetCounselor :one
SELECT id, camp_id, counselor_name, junior_counselor, counselor_enabled
FROM counselors
WHERE id = $1;

-- name: CreateCounselor :one
INSERT INTO counselors (camp_id, counselor_name, junior_counselor)
VALUES ($1, $2, $3)
RETURNING id, camp_id, counselor_name, junior_counselor, counselor_enabled;

-- name: UpdateCounselor :one
UPDATE counselors
SET counselor_name = $2,
    junior_counselor = $3,
    counselor_enabled = $4
WHERE id = $1
RETURNING id, camp_id, counselor_name, junior_counselor, counselor_enabled;

-- name: DeleteCounselor :execrows
DELETE FROM counselors
WHERE id = $1;
