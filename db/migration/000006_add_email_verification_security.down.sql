-- Drop functions
DROP FUNCTION IF EXISTS log_security_event(BIGINT, VARCHAR(50), TEXT, INET, TEXT, JSONB);
DROP FUNCTION IF EXISTS unlock_expired_accounts();
DROP FUNCTION IF EXISTS cleanup_expired_tokens();

-- Drop indexes
DROP INDEX IF EXISTS idx_user_sessions_expires;
DROP INDEX IF EXISTS idx_user_sessions_refresh_token;
DROP INDEX IF EXISTS idx_user_sessions_token;
DROP INDEX IF EXISTS idx_user_sessions_user;

DROP INDEX IF EXISTS idx_account_lockouts_locked_until;
DROP INDEX IF EXISTS idx_account_lockouts_user;

DROP INDEX IF EXISTS idx_security_events_ip;
DROP INDEX IF EXISTS idx_security_events_type_created;
DROP INDEX IF EXISTS idx_security_events_user_created;

DROP INDEX IF EXISTS idx_password_reset_tokens_expires;
DROP INDEX IF EXISTS idx_password_reset_tokens_user;
DROP INDEX IF EXISTS idx_password_reset_tokens_token;

DROP INDEX IF EXISTS idx_email_verification_tokens_expires;
DROP INDEX IF EXISTS idx_email_verification_tokens_user_type;
DROP INDEX IF EXISTS idx_email_verification_tokens_token;

-- Remove columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;

-- Drop tables
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_2fa;
DROP TABLE IF EXISTS account_lockouts;
DROP TABLE IF EXISTS security_events;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS email_verification_tokens;
