-- name: GetUnreadMessages :many
SELECT * FROM unread_messages
WHERE user_id = $1 AND workspace_id = $2
ORDER BY updated_at DESC;

-- name: GetChannelUnreadCount :one
SELECT COALESCE(unread_count, 0) as unread_count
FROM unread_messages
WHERE user_id = $1 AND channel_id = $2;

-- name: GetDirectMessageUnreadCount :one
SELECT COALESCE(SUM(unread_count), 0) as total_unread
FROM unread_messages
WHERE user_id = $1 AND channel_id IS NULL;

-- name: MarkChannelAsRead :exec
INSERT INTO unread_messages (user_id, workspace_id, channel_id, last_read_message_id, unread_count, last_read_at, updated_at)
VALUES ($1, $2, $3, $4, 0, now(), now())
ON CONFLICT (user_id, channel_id)
DO UPDATE SET 
    last_read_message_id = $4,
    unread_count = 0,
    last_read_at = now(),
    updated_at = now();

-- name: MarkDirectMessagesAsRead :exec
INSERT INTO unread_messages (user_id, workspace_id, channel_id, last_read_message_id, unread_count, last_read_at, updated_at)
VALUES ($1, $2, NULL, $3, 0, now(), now())
ON CONFLICT (user_id, channel_id)
DO UPDATE SET 
    last_read_message_id = $3,
    unread_count = 0,
    last_read_at = now(),
    updated_at = now();

-- name: GetWorkspaceUnreadCount :one
SELECT COALESCE(SUM(unread_count), 0) as total_unread
FROM unread_messages
WHERE user_id = $1 AND workspace_id = $2;

-- name: ResetUnreadCount :exec
UPDATE unread_messages 
SET unread_count = 0, last_read_at = now(), updated_at = now()
WHERE user_id = $1 AND channel_id = $2;
