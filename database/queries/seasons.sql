-- name: ListSeasons :many
SELECT id, camp_id, season_name
FROM seasons
WHERE camp_id = $1
ORDER BY season_name;

-- name: GetSeason :one
SELECT id, camp_id, season_name
FROM seasons
WHERE id = $1;

-- name: CreateSeason :one
INSERT INTO seasons (camp_id, season_name)
VALUES ($1, $2)
RETURNING id, camp_id, season_name;

-- name: UpdateSeason :one
UPDATE seasons
SET season_name = $2
WHERE id = $1
RETURNING id, camp_id, season_name;

-- name: DeleteSeason :execrows
DELETE FROM seasons
WHERE id = $1;
