-- name: ListCertifications :many
SELECT id, camp_id, certification_name, archived
FROM certifications
WHERE camp_id = $1 AND archived = false
ORDER BY certification_name;

-- name: ListAllCertifications :many
SELECT id, camp_id, certification_name, archived
FROM certifications
WHERE camp_id = $1
ORDER BY certification_name;

-- name: ListArchivedCertifications :many
SELECT id, camp_id, certification_name, archived
FROM certifications
WHERE camp_id = $1 AND archived = true
ORDER BY certification_name;

-- name: GetCertification :one
SELECT id, camp_id, certification_name, archived
FROM certifications
WHERE id = $1 AND camp_id = $2;

-- name: CreateCertification :one
INSERT INTO certifications (camp_id, certification_name)
VALUES ($1, $2)
RETURNING id, camp_id, certification_name, archived;

-- name: UpdateCertification :one
UPDATE certifications
SET certification_name = $3
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, certification_name, archived;

-- name: DeleteCertification :execrows
DELETE FROM certifications
WHERE id = $1 AND camp_id = $2;

-- name: ArchiveCertification :execrows
UPDATE certifications
SET archived = true
WHERE id = $1 AND camp_id = $2;

-- name: UnarchiveCertification :execrows
UPDATE certifications
SET archived = false
WHERE id = $1 AND camp_id = $2;
