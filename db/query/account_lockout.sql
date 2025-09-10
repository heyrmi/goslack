-- name: GetAccountLockout :one
SELECT * FROM account_lockouts WHERE user_id = $1;

-- name: CreateAccountLockout :one
INSERT INTO account_lockouts (user_id, failed_attempts, last_failed_attempt)
VALUES ($1, 1, now())
RETURNING *;

-- name: IncrementFailedAttempts :one
UPDATE account_lockouts 
SET failed_attempts = failed_attempts + 1, 
    last_failed_attempt = now(),
    updated_at = now()
WHERE user_id = $1 
RETURNING *;

-- name: LockAccount :exec
UPDATE account_lockouts 
SET locked_until = $2, updated_at = now()
WHERE user_id = $1;

-- name: UnlockAccount :exec
UPDATE account_lockouts 
SET locked_until = NULL, 
    failed_attempts = 0, 
    updated_at = now()
WHERE user_id = $1;

-- name: ResetFailedAttempts :exec
UPDATE account_lockouts 
SET failed_attempts = 0, 
    last_failed_attempt = NULL,
    updated_at = now()
WHERE user_id = $1;

-- name: UnlockExpiredAccounts :exec
UPDATE account_lockouts 
SET locked_until = NULL, failed_attempts = 0, updated_at = now()
WHERE locked_until IS NOT NULL AND locked_until < now();

-- name: IsAccountLocked :one
SELECT locked_until IS NOT NULL AND locked_until > now() as is_locked
FROM account_lockouts 
WHERE user_id = $1;
