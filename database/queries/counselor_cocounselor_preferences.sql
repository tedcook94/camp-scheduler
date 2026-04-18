-- name: ListCounselorCocounselorPreferences :many
SELECT id, camp_id, counselor_id, session_id, preferred_counselor_id, rank
FROM counselor_cocounselor_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: CreateCounselorCocounselorPreference :one
INSERT INTO counselor_cocounselor_preferences (camp_id, counselor_id, session_id, preferred_counselor_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, counselor_id, session_id, preferred_counselor_id, rank;

-- name: DeleteAllCounselorCocounselorPreferences :exec
DELETE FROM counselor_cocounselor_preferences
WHERE counselor_id = $1 AND session_id = $2 AND camp_id = $3;

-- name: ListSessionCocounselorPreferences :many
SELECT counselor_id, preferred_counselor_id, rank
FROM counselor_cocounselor_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY counselor_id, rank;
