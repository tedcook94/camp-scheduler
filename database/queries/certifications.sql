-- name: ListCertifications :many
SELECT id, camp_id, certification_name
FROM certifications
WHERE camp_id = $1
ORDER BY certification_name;

-- name: GetCertification :one
SELECT id, camp_id, certification_name
FROM certifications
WHERE id = $1 AND camp_id = $2;

-- name: CreateCertification :one
INSERT INTO certifications (camp_id, certification_name)
VALUES ($1, $2)
RETURNING id, camp_id, certification_name;

-- name: UpdateCertification :one
UPDATE certifications
SET certification_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, certification_name;

-- name: DeleteCertification :execrows
DELETE FROM certifications
WHERE id = $1 AND camp_id = $2;
