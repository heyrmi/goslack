-- name: PinMessage :one
INSERT INTO pinned_messages (message_id, channel_id, pinned_by)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UnpinMessage :exec
DELETE FROM pinned_messages WHERE message_id = $1;

-- name: GetPinnedMessages :many
SELECT pm.*, m.content, m.created_at as message_created_at,
       u.first_name as sender_first_name, u.last_name as sender_last_name,
       pu.first_name as pinned_by_first_name, pu.last_name as pinned_by_last_name
FROM pinned_messages pm
JOIN messages m ON pm.message_id = m.id
JOIN users u ON m.sender_id = u.id
JOIN users pu ON pm.pinned_by = pu.id
WHERE pm.channel_id = $1
ORDER BY pm.pinned_at DESC;

-- name: IsMessagePinned :one
SELECT EXISTS(
    SELECT 1 FROM pinned_messages WHERE message_id = $1
) as is_pinned;
