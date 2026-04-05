-- name: ListCamperFriendPreferences :many
SELECT id, camp_id, camper_id, session_id, preferred_camper_id, rank
FROM camper_friend_preferences
WHERE camper_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: GetCamperFriendPreference :one
SELECT id, camp_id, camper_id, session_id, preferred_camper_id, rank
FROM camper_friend_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND camper_id = $4;

-- name: CreateCamperFriendPreference :one
INSERT INTO camper_friend_preferences (camp_id, camper_id, session_id, preferred_camper_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, camper_id, session_id, preferred_camper_id, rank;

-- name: UpdateCamperFriendPreference :one
UPDATE camper_friend_preferences
SET preferred_camper_id = $5,
    rank = $6
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND camper_id = $4
RETURNING id, camp_id, camper_id, session_id, preferred_camper_id, rank;

-- name: DeleteCamperFriendPreference :execrows
DELETE FROM camper_friend_preferences
WHERE id = $1 AND camp_id = $2 AND session_id = $3 AND camper_id = $4;

-- name: ListSessionCamperFriendPreferences :many
SELECT camper_id, preferred_camper_id, rank
FROM camper_friend_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY camper_id, rank;
