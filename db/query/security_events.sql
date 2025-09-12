-- name: CreateSecurityEvent :one
INSERT INTO security_events (
    user_id, event_type, description, ip_address, user_agent, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetUserSecurityEvents :many
SELECT * FROM security_events 
WHERE user_id = $1 
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: GetSecurityEventsByType :many
SELECT * FROM security_events 
WHERE event_type = $1 
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3;

-- name: GetRecentSecurityEvents :many
SELECT * FROM security_events 
WHERE created_at >= $1 
ORDER BY created_at DESC 
LIMIT $2;

-- name: CleanupOldSecurityEvents :exec
DELETE FROM security_events 
WHERE created_at < $1;
