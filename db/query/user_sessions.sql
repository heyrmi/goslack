-- name: CreateUserSession :one
INSERT INTO user_sessions (
    user_id, session_token, refresh_token, expires_at, 
    ip_address, user_agent, device_info
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetUserSession :one
SELECT * FROM user_sessions 
WHERE session_token = $1 AND is_active = true AND expires_at > now();

-- name: GetUserSessionByRefreshToken :one
SELECT * FROM user_sessions 
WHERE refresh_token = $1 AND is_active = true AND expires_at > now();

-- name: UpdateSessionLastUsed :exec
UPDATE user_sessions 
SET last_used_at = now() 
WHERE session_token = $1;

-- name: DeactivateSession :exec
UPDATE user_sessions 
SET is_active = false 
WHERE session_token = $1;

-- name: DeactivateUserSessions :exec
UPDATE user_sessions 
SET is_active = false 
WHERE user_id = $1 AND is_active = true;

-- name: DeactivateExpiredSessions :exec
UPDATE user_sessions 
SET is_active = false 
WHERE expires_at < now() AND is_active = true;

-- name: CleanupOldSessions :exec
DELETE FROM user_sessions 
WHERE is_active = false AND last_used_at < $1;

-- name: GetUserActiveSessions :many
SELECT * FROM user_sessions 
WHERE user_id = $1 AND is_active = true 
ORDER BY last_used_at DESC;
