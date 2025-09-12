-- name: CreateScheduledMessage :one
INSERT INTO scheduled_messages (
    user_id, workspace_id, channel_id, receiver_id, thread_id,
    content, content_type, scheduled_for
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetScheduledMessage :one
SELECT * FROM scheduled_messages
WHERE id = $1 AND user_id = $2;

-- name: GetUserScheduledMessages :many
SELECT sm.*, c.name as channel_name, u.first_name, u.last_name
FROM scheduled_messages sm
LEFT JOIN channels c ON sm.channel_id = c.id
LEFT JOIN users u ON sm.receiver_id = u.id
WHERE sm.user_id = $1 AND sm.workspace_id = $2
  AND sm.sent_at IS NULL AND sm.cancelled_at IS NULL
ORDER BY sm.scheduled_for ASC
LIMIT $3 OFFSET $4;

-- name: GetPendingScheduledMessages :many
SELECT * FROM scheduled_messages
WHERE scheduled_for <= now()
  AND sent_at IS NULL 
  AND cancelled_at IS NULL
ORDER BY scheduled_for ASC
LIMIT $1;

-- name: MarkScheduledMessageAsSent :exec
UPDATE scheduled_messages
SET sent_at = now()
WHERE id = $1;

-- name: CancelScheduledMessage :exec
UPDATE scheduled_messages
SET cancelled_at = now()
WHERE id = $1 AND user_id = $2 AND sent_at IS NULL;

-- name: UpdateScheduledMessage :one
UPDATE scheduled_messages
SET 
    content = $3,
    content_type = $4,
    scheduled_for = $5
WHERE id = $1 AND user_id = $2 AND sent_at IS NULL AND cancelled_at IS NULL
RETURNING *;

-- name: DeleteScheduledMessage :exec
DELETE FROM scheduled_messages
WHERE id = $1 AND user_id = $2 AND sent_at IS NULL;

-- name: CleanupOldScheduledMessages :exec
DELETE FROM scheduled_messages
WHERE (sent_at IS NOT NULL OR cancelled_at IS NOT NULL)
  AND created_at < $1;

-- name: GetScheduledMessagesStats :one
SELECT 
    COUNT(*) FILTER (WHERE sent_at IS NULL AND cancelled_at IS NULL) as pending_count,
    COUNT(*) FILTER (WHERE sent_at IS NOT NULL) as sent_count,
    COUNT(*) FILTER (WHERE cancelled_at IS NOT NULL) as cancelled_count
FROM scheduled_messages
WHERE user_id = $1 AND workspace_id = $2;
