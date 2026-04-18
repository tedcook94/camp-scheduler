-- name: ListCamperFriendPreferences :many
SELECT id, camp_id, camper_id, session_id, preferred_camper_id, rank
FROM camper_friend_preferences
WHERE camper_id = $1 AND session_id = $2 AND camp_id = $3
ORDER BY rank;

-- name: CreateCamperFriendPreference :one
INSERT INTO camper_friend_preferences (camp_id, camper_id, session_id, preferred_camper_id, rank)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, camp_id, camper_id, session_id, preferred_camper_id, rank;

-- name: DeleteAllCamperFriendPreferences :exec
DELETE FROM camper_friend_preferences
WHERE camper_id = $1 AND session_id = $2 AND camp_id = $3;

-- name: ListSessionCamperFriendPreferences :many
SELECT camper_id, preferred_camper_id, rank
FROM camper_friend_preferences
WHERE session_id = $1 AND camp_id = $2
ORDER BY camper_id, rank;
