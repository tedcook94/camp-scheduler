-- name: ListSessionsByCamp :many
SELECT id
FROM sessions
WHERE camp_id = $1;

-- name: ListSessionsByCounselorRoster :many
SELECT DISTINCT sc.session_id AS id
FROM session_counselors sc
WHERE sc.camp_id = $1 AND sc.counselor_id = $2;

-- name: ListSessionsByCamperEnrollment :many
SELECT DISTINCT cse.session_id AS id
FROM camper_session_enrollments cse
WHERE cse.camp_id = $1 AND cse.camper_id = $2;
