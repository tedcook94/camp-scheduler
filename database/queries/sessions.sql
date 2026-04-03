-- name: ListSessions :many
SELECT id, camp_id, season_id, session_name, previous_session
FROM sessions
WHERE camp_id = $1
ORDER BY session_name;

-- name: GetSession :one
SELECT id, camp_id, season_id, session_name, previous_session
FROM sessions
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (camp_id, season_id, session_name, previous_session)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, season_id, session_name, previous_session;

-- name: UpdateSession :one
UPDATE sessions
SET season_id = $2,
    session_name = $3,
    previous_session = $4
WHERE id = $1
RETURNING id, camp_id, season_id, session_name, previous_session;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE id = $1;
