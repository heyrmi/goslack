-- name: CreateUser2FA :one
INSERT INTO user_2fa (user_id, secret, backup_codes)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser2FA :one
SELECT * FROM user_2fa WHERE user_id = $1;

-- name: EnableUser2FA :exec
UPDATE user_2fa 
SET enabled = true, verified_at = now(), updated_at = now()
WHERE user_id = $1;

-- name: DisableUser2FA :exec
UPDATE user_2fa 
SET enabled = false, updated_at = now()
WHERE user_id = $1;

-- name: UpdateUser2FABackupCodes :exec
UPDATE user_2fa 
SET backup_codes = $2, updated_at = now()
WHERE user_id = $1;

-- name: DeleteUser2FA :exec
DELETE FROM user_2fa WHERE user_id = $1;
