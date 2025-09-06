-- name: CreateChannel :one
INSERT INTO channels (
    workspace_id,
    name,
    is_private,
    created_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetChannel :one
SELECT * FROM channels
WHERE id = $1 LIMIT 1;

-- name: GetChannelByID :one
SELECT * FROM channels
WHERE id = $1 LIMIT 1;

-- name: ListChannelsByWorkspace :many
SELECT * FROM channels
WHERE workspace_id = $1
ORDER BY created_at ASC
LIMIT $2
OFFSET $3;

-- name: ListPublicChannelsByWorkspace :many
SELECT * FROM channels
WHERE workspace_id = $1 AND is_private = false
ORDER BY created_at ASC
LIMIT $2
OFFSET $3;

-- name: UpdateChannel :one
UPDATE channels
SET 
    name = $2,
    is_private = $3
WHERE id = $1
RETURNING *;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE id = $1;

-- name: GetChannelWithCreator :one
SELECT 
    c.*,
    u.first_name as creator_first_name,
    u.last_name as creator_last_name,
    u.email as creator_email
FROM channels c
JOIN users u ON c.created_by = u.id
WHERE c.id = $1
LIMIT 1;
