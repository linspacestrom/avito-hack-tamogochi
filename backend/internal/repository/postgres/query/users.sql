-- name: CreateUser :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, display_name, password_hash, status, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, display_name, password_hash, status, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, display_name, password_hash, status, created_at, updated_at
FROM users
WHERE id = $1;
