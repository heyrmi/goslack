-- name: CreateChannelMessage :one
INSERT INTO messages (
    workspace_id,
    channel_id,
    sender_id,
    content,
    content_type,
    message_type
) VALUES (
    $1, $2, $3, $4, $5, 'channel'
)
RETURNING *;

-- name: CreateDirectMessage :one
INSERT INTO messages (
    workspace_id,
    sender_id,
    receiver_id,
    content,
    content_type,
    message_type
) VALUES (
    $1, $2, $3, $4, $5, 'direct'
)
RETURNING *;

-- name: GetChannelMessages :many
SELECT 
    m.*,
    u.first_name as sender_first_name,
    u.last_name as sender_last_name,
    u.email as sender_email
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.channel_id = $1 
    AND m.workspace_id = $2 
    AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
LIMIT $3
OFFSET $4;

-- name: GetDirectMessagesBetweenUsers :many
SELECT 
    m.*,
    u.first_name as sender_first_name,
    u.last_name as sender_last_name,
    u.email as sender_email
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.workspace_id = $1 
    AND m.message_type = 'direct'
    AND m.deleted_at IS NULL
    AND (
        (m.sender_id = $2 AND m.receiver_id = $3) OR
        (m.sender_id = $3 AND m.receiver_id = $2)
    )
ORDER BY m.created_at DESC
LIMIT $4
OFFSET $5;

-- name: GetMessageByID :one
SELECT 
    m.*,
    u.first_name as sender_first_name,
    u.last_name as sender_last_name,
    u.email as sender_email
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.id = $1 AND m.deleted_at IS NULL;

-- name: UpdateMessageContent :one
UPDATE messages
SET 
    content = $2,
    edited_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteMessage :exec
UPDATE messages
SET deleted_at = now()
WHERE id = $1;

-- name: GetRecentWorkspaceMessages :many
SELECT 
    m.*,
    u.first_name as sender_first_name,
    u.last_name as sender_last_name,
    u.email as sender_email
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.workspace_id = $1 
    AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
LIMIT $2
OFFSET $3;

-- name: CheckMessageAuthor :one
SELECT sender_id FROM messages
WHERE id = $1 AND deleted_at IS NULL;