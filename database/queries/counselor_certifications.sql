-- name: ListCounselorCertifications :many
SELECT cc.id, cc.camp_id, cc.counselor_id, cc.certification_id, c.certification_name
FROM counselor_certifications cc
JOIN certifications c ON c.id = cc.certification_id
WHERE cc.counselor_id = $1 AND cc.camp_id = $2
ORDER BY c.certification_name;

-- name: CreateCounselorCertification :one
INSERT INTO counselor_certifications (camp_id, counselor_id, certification_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, counselor_id, certification_id;

-- name: DeleteCounselorCertification :execrows
DELETE FROM counselor_certifications
WHERE id = $1 AND camp_id = $2 AND counselor_id = $3;

-- name: ListSessionCounselorCertifications :many
SELECT cc.counselor_id, cc.certification_id
FROM counselor_certifications cc
JOIN counselors co ON co.id = cc.counselor_id AND co.camp_id = cc.camp_id
WHERE cc.camp_id = $1 AND co.counselor_enabled = true
ORDER BY cc.counselor_id, cc.certification_id;
