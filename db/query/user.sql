-- name: CreateUser :one
INSERT INTO users (
    organization_id,
    email,
    first_name,
    last_name,
    hashed_password
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
WHERE organization_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateUserPassword :one
UPDATE users
SET
    hashed_password = $2,
    password_changed_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE users
SET
    first_name = $2,
    last_name = $3
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
