-- name: CreateMessageMention :one
INSERT INTO message_mentions (message_id, mentioned_user_id, mention_type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMessageMentions :many
SELECT mm.*, u.first_name, u.last_name, u.email
FROM message_mentions mm
LEFT JOIN users u ON mm.mentioned_user_id = u.id
WHERE mm.message_id = $1
ORDER BY mm.created_at ASC;

-- name: GetUserMentions :many
SELECT mm.message_id, mm.mention_type, mm.created_at,
       m.content, m.created_at as message_created_at,
       u.first_name as sender_first_name, u.last_name as sender_last_name,
       c.name as channel_name
FROM message_mentions mm
JOIN messages m ON mm.message_id = m.id
JOIN users u ON m.sender_id = u.id
LEFT JOIN channels c ON m.channel_id = c.id
WHERE mm.mentioned_user_id = $1 
   OR (mm.mention_type IN ('channel', 'here', 'everyone') AND m.workspace_id = $2)
ORDER BY mm.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetUnreadMentions :many
SELECT mm.message_id, mm.mention_type, mm.created_at,
       m.content, m.created_at as message_created_at,
       u.first_name as sender_first_name, u.last_name as sender_last_name,
       c.name as channel_name
FROM message_mentions mm
JOIN messages m ON mm.message_id = m.id
JOIN users u ON m.sender_id = u.id
LEFT JOIN channels c ON m.channel_id = c.id
LEFT JOIN unread_messages um ON (
    (m.channel_id IS NOT NULL AND um.channel_id = m.channel_id AND um.user_id = $1) OR
    (m.receiver_id = $1 AND um.channel_id IS NULL AND um.user_id = $1)
)
WHERE (mm.mentioned_user_id = $1 OR (mm.mention_type IN ('channel', 'here', 'everyone') AND m.workspace_id = $2))
  AND (um.last_read_message_id IS NULL OR m.id > um.last_read_message_id)
ORDER BY mm.created_at DESC
LIMIT $3;
