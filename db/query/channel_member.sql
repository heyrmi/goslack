-- name: AddChannelMember :one
INSERT INTO channel_members (
    channel_id,
    user_id,
    added_by,
    role
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: RemoveChannelMember :exec
DELETE FROM channel_members
WHERE channel_id = $1 AND user_id = $2;

-- name: GetChannelMembers :many
SELECT 
    cm.*,
    u.first_name,
    u.last_name,
    u.email
FROM channel_members cm
JOIN users u ON cm.user_id = u.id
WHERE cm.channel_id = $1
ORDER BY cm.joined_at ASC
LIMIT $2
OFFSET $3;

-- name: CheckChannelMembership :one
SELECT role FROM channel_members
WHERE channel_id = $1 AND user_id = $2;

-- name: GetUserChannels :many
SELECT 
    c.*
FROM channels c
JOIN channel_members cm ON c.id = cm.channel_id
WHERE cm.user_id = $1 AND c.workspace_id = $2
ORDER BY c.created_at ASC;

-- name: IsChannelMember :one
SELECT EXISTS(
    SELECT 1 FROM channel_members
    WHERE channel_id = $1 AND user_id = $2
);