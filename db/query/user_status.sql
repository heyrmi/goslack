-- name: UpsertUserStatus :one
INSERT INTO user_status (
    user_id,
    workspace_id,
    status,
    custom_status,
    updated_at
) VALUES (
    $1, $2, $3, $4, now()
)
ON CONFLICT (user_id) DO UPDATE SET
    status = EXCLUDED.status,
    custom_status = EXCLUDED.custom_status,
    updated_at = now()
RETURNING *;

-- name: GetUserStatus :one
SELECT * FROM user_status
WHERE user_id = $1 AND workspace_id = $2;

-- name: GetWorkspaceUserStatuses :many
SELECT 
    us.*,
    u.first_name,
    u.last_name,
    u.email
FROM user_status us
JOIN users u ON us.user_id = u.id
WHERE us.workspace_id = $1
ORDER BY us.updated_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateLastActivity :exec
UPDATE user_status
SET 
    last_activity_at = now(),
    last_seen_at = now(),
    updated_at = now()
WHERE user_id = $1 AND workspace_id = $2;

-- name: SetUsersOfflineAfterInactivity :exec
UPDATE user_status
SET 
    status = 'offline',
    updated_at = now()
WHERE last_activity_at < $1 AND status != 'offline';

-- name: GetOnlineUsersInWorkspace :many
SELECT 
    us.*,
    u.first_name,
    u.last_name,
    u.email
FROM user_status us
JOIN users u ON us.user_id = u.id
WHERE us.workspace_id = $1 
    AND us.status IN ('online', 'away', 'busy')
ORDER BY us.updated_at DESC;