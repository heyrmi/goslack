-- name: SearchMessages :many
SELECT m.id, m.workspace_id, m.channel_id, m.sender_id, m.receiver_id, 
       m.content, m.message_type, m.thread_id, m.created_at, m.content_type,
       u.first_name, u.last_name,
       c.name as channel_name,
       ts_rank(msi.content_vector, plainto_tsquery($4)) as rank
FROM message_search_index msi
JOIN messages m ON msi.message_id = m.id
JOIN users u ON m.sender_id = u.id
LEFT JOIN channels c ON m.channel_id = c.id
WHERE msi.workspace_id = $1
  AND ($2::BIGINT IS NULL OR msi.channel_id = $2)
  AND ($3::BIGINT IS NULL OR msi.user_id = $3)
  AND msi.content_vector @@ plainto_tsquery($4)
  AND m.deleted_at IS NULL
ORDER BY rank DESC, m.created_at DESC
LIMIT $5 OFFSET $6;

-- name: SearchMessagesInThread :many
SELECT m.id, m.workspace_id, m.channel_id, m.sender_id, m.receiver_id, 
       m.content, m.message_type, m.thread_id, m.created_at, m.content_type,
       u.first_name, u.last_name,
       ts_rank(msi.content_vector, plainto_tsquery($3)) as rank
FROM message_search_index msi
JOIN messages m ON msi.message_id = m.id
JOIN users u ON m.sender_id = u.id
WHERE (m.id = $1 OR m.thread_id = $1)
  AND msi.workspace_id = $2
  AND msi.content_vector @@ plainto_tsquery($3)
  AND m.deleted_at IS NULL
ORDER BY rank DESC, m.created_at ASC
LIMIT $4 OFFSET $5;
