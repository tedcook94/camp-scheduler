-- name: ListSessions :many
SELECT id, camp_id, season_id, session_name, previous_session, archived
FROM sessions
WHERE camp_id = $1
ORDER BY session_name;

-- name: GetSession :one
SELECT id, camp_id, season_id, session_name, previous_session, archived
FROM sessions
WHERE id = $1 AND camp_id = $2;

-- name: CreateSession :one
INSERT INTO sessions (camp_id, season_id, session_name, previous_session)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, season_id, session_name, previous_session, archived;

-- name: UpdateSession :one
UPDATE sessions
SET season_id = $3,
    session_name = $4,
    previous_session = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, season_id, session_name, previous_session, archived;

-- name: HasDependentSessions :one
SELECT EXISTS (
    SELECT 1 FROM sessions
    WHERE previous_session = $1 AND camp_id = $2
) AS has_dependents;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE id = $1 AND camp_id = $2;
