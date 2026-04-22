-- name: ListSessionCounselors :many
SELECT sc.id, sc.camp_id, sc.session_id, sc.counselor_id,
       c.counselor_name, c.junior_counselor, c.counselor_enabled
FROM session_counselors sc
JOIN counselors c ON c.id = sc.counselor_id AND c.camp_id = sc.camp_id
WHERE sc.session_id = $1 AND sc.camp_id = $2
ORDER BY c.counselor_name;

-- name: ListSessionCounselorIDs :many
SELECT counselor_id
FROM session_counselors
WHERE session_id = $1 AND camp_id = $2;

-- name: GetSessionCounselor :one
SELECT id, camp_id, session_id, counselor_id
FROM session_counselors
WHERE session_id = $1 AND camp_id = $2 AND counselor_id = $3;

-- name: AddSessionCounselor :one
INSERT INTO session_counselors (camp_id, session_id, counselor_id)
VALUES ($1, $2, $3)
RETURNING id, camp_id, session_id, counselor_id;

-- name: RemoveSessionCounselor :execrows
DELETE FROM session_counselors
WHERE session_id = $1 AND camp_id = $2 AND counselor_id = $3;

-- name: DeleteAllSessionCounselors :exec
DELETE FROM session_counselors
WHERE session_id = $1 AND camp_id = $2;

-- name: RemoveCounselorFromAllSessions :exec
DELETE FROM session_counselors
WHERE camp_id = $1 AND counselor_id = $2;

-- name: IsCounselorInSession :one
SELECT EXISTS (
    SELECT 1 FROM session_counselors
    WHERE session_id = $1 AND camp_id = $2 AND counselor_id = $3
) AS exists;
