-- name: GetUserByID :one
SELECT id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at
FROM users
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at;
