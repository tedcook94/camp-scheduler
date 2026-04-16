-- name: GetUserByID :one
SELECT id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at, token_version
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at, token_version
FROM users
WHERE username = $1;

-- name: GetUserTokenVersion :one
SELECT token_version
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, camp_id, username, email, password_hash, first_name, last_name, role, created_at, updated_at, token_version;

-- name: ListUsers :many
SELECT id, camp_id, username, email, first_name, last_name, role, created_at, updated_at
FROM users
ORDER BY created_at;

-- name: ListUsersByCamp :many
SELECT id, camp_id, username, email, first_name, last_name, role, created_at, updated_at
FROM users
WHERE camp_id = $1
ORDER BY created_at;

-- name: UpdateUser :one
UPDATE users
SET camp_id    = $2,
    username   = $3,
    email      = $4,
    first_name = $5,
    last_name  = $6,
    role       = $7,
    updated_at = now()
WHERE id = $1
RETURNING id, camp_id, username, email, first_name, last_name, role, created_at, updated_at;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash  = $2,
    token_version  = token_version + 1,
    updated_at     = now()
WHERE id = $1;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = $1;
