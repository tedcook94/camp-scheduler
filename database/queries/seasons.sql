-- name: ListSeasons :many
SELECT id, camp_id, season_name, start_date, end_date, archived
FROM seasons
WHERE camp_id = $1 AND archived = false
ORDER BY start_date DESC, id DESC;

-- name: ListAllSeasons :many
SELECT id, camp_id, season_name, start_date, end_date, archived
FROM seasons
WHERE camp_id = $1
ORDER BY start_date DESC, id DESC;

-- name: ListArchivedSeasons :many
SELECT id, camp_id, season_name, start_date, end_date, archived
FROM seasons
WHERE camp_id = $1 AND archived = true
ORDER BY start_date DESC, id DESC;

-- name: GetSeason :one
SELECT id, camp_id, season_name, start_date, end_date, archived
FROM seasons
WHERE id = $1 AND camp_id = $2;

-- name: CreateSeason :one
INSERT INTO seasons (camp_id, season_name, start_date, end_date)
VALUES ($1, $2, $3, $4)
RETURNING id, camp_id, season_name, start_date, end_date, archived;

-- name: UpdateSeason :one
UPDATE seasons
SET season_name = $3, start_date = $4, end_date = $5
WHERE id = $1 AND camp_id = $2
RETURNING id, camp_id, season_name, start_date, end_date, archived;

-- name: DeleteSeason :execrows
DELETE FROM seasons
WHERE id = $1 AND camp_id = $2;

-- name: GetMostRecentSeason :one
SELECT id, camp_id, season_name, start_date, end_date, archived
FROM seasons
WHERE camp_id = $1 AND archived = false
ORDER BY start_date DESC, id DESC
LIMIT 1;
