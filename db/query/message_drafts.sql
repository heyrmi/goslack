-- name: SaveMessageDraft :one
INSERT INTO message_drafts (user_id, workspace_id, channel_id, receiver_id, thread_id, content, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (user_id, channel_id, receiver_id, thread_id)
DO UPDATE SET content = EXCLUDED.content, updated_at = now()
RETURNING *;

-- name: GetMessageDraft :one
SELECT * FROM message_drafts
WHERE user_id = $1 
  AND (channel_id = $2 OR (channel_id IS NULL AND $2 IS NULL))
  AND (receiver_id = $3 OR (receiver_id IS NULL AND $3 IS NULL))
  AND (thread_id = $4 OR (thread_id IS NULL AND $4 IS NULL));

-- name: DeleteMessageDraft :exec
DELETE FROM message_drafts
WHERE user_id = $1 
  AND (channel_id = $2 OR (channel_id IS NULL AND $2 IS NULL))
  AND (receiver_id = $3 OR (receiver_id IS NULL AND $3 IS NULL))
  AND (thread_id = $4 OR (thread_id IS NULL AND $4 IS NULL));

-- name: GetUserDrafts :many
SELECT md.*, c.name as channel_name, u.first_name, u.last_name
FROM message_drafts md
LEFT JOIN channels c ON md.channel_id = c.id
LEFT JOIN users u ON md.receiver_id = u.id
WHERE md.user_id = $1 AND md.workspace_id = $2
ORDER BY md.updated_at DESC;

-- name: CleanupOldDrafts :exec
DELETE FROM message_drafts 
WHERE updated_at < $1;
