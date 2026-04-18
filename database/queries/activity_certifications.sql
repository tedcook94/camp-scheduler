-- name: ListActivityCertifications :many
SELECT ac.id, ac.camp_id, ac.activity_id, ac.certification_id, c.certification_name
FROM activity_certifications ac
JOIN certifications c ON c.id = ac.certification_id
WHERE ac.activity_id = $1 AND ac.camp_id = $2
ORDER BY c.certification_name;

-- name: ListAllActivityCertifications :many
SELECT ac.id, ac.camp_id, ac.activity_id, ac.certification_id, c.certification_name
FROM activity_certifications ac
JOIN certifications c ON c.id = ac.certification_id
WHERE ac.camp_id = $1
ORDER BY ac.activity_id, c.certification_name;

-- name: CreateActivityCertification :one
INSERT INTO activity_certifications (camp_id, activity_id, certification_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, activity_id, certification_id;

-- name: DeleteActivityCertification :execrows
DELETE FROM activity_certifications
WHERE id = $1 AND camp_id = $2 AND activity_id = $3;
