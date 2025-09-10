-- name: AddMessageReaction :one
INSERT INTO message_reactions (message_id, user_id, emoji)
VALUES ($1, $2, $3)
ON CONFLICT (message_id, user_id, emoji) DO NOTHING
RETURNING *;

-- name: RemoveMessageReaction :exec
DELETE FROM message_reactions 
WHERE message_id = $1 AND user_id = $2 AND emoji = $3;

-- name: GetMessageReactions :many
SELECT mr.*, u.first_name, u.last_name 
FROM message_reactions mr
JOIN users u ON mr.user_id = u.id
WHERE mr.message_id = $1
ORDER BY mr.created_at ASC;

-- name: GetMessageReactionCounts :many
SELECT emoji, COUNT(*) as count
FROM message_reactions
WHERE message_id = $1
GROUP BY emoji
ORDER BY count DESC, emoji ASC;

-- name: HasUserReacted :one
SELECT EXISTS(
    SELECT 1 FROM message_reactions 
    WHERE message_id = $1 AND user_id = $2 AND emoji = $3
) as has_reacted;

-- name: GetUserReactionsForMessage :many
SELECT emoji FROM message_reactions
WHERE message_id = $1 AND user_id = $2
ORDER BY created_at ASC;
