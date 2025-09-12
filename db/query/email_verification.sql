-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (
    user_id, token, email, token_type, expires_at, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetEmailVerificationToken :one
SELECT * FROM email_verification_tokens 
WHERE token = $1 AND used_at IS NULL AND expires_at > now();

-- name: UseEmailVerificationToken :exec
UPDATE email_verification_tokens 
SET used_at = now() 
WHERE token = $1 AND used_at IS NULL;

-- name: DeleteExpiredEmailVerificationTokens :exec
DELETE FROM email_verification_tokens 
WHERE expires_at < now() AND used_at IS NULL;

-- name: GetUserEmailVerificationTokens :many
SELECT * FROM email_verification_tokens 
WHERE user_id = $1 AND token_type = $2 
ORDER BY created_at DESC;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    user_id, token, expires_at, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetPasswordResetToken :one
SELECT * FROM password_reset_tokens 
WHERE token = $1 AND used_at IS NULL AND expires_at > now();

-- name: UsePasswordResetToken :exec
UPDATE password_reset_tokens 
SET used_at = now() 
WHERE token = $1 AND used_at IS NULL;

-- name: DeleteExpiredPasswordResetTokens :exec
DELETE FROM password_reset_tokens 
WHERE expires_at < now() AND used_at IS NULL;

-- name: DeleteUserPasswordResetTokens :exec
DELETE FROM password_reset_tokens 
WHERE user_id = $1 AND used_at IS NULL;
