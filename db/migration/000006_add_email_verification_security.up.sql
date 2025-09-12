-- Email verification and authentication security features

-- Email verification tokens
CREATE TABLE email_verification_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    token_type VARCHAR(50) NOT NULL CHECK (token_type IN ('email_verification', 'password_reset', 'email_change')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    ip_address INET,
    user_agent TEXT
);

-- Password reset tokens (separate table for better security)
CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    ip_address INET,
    user_agent TEXT
);

-- Account security events
CREATE TABLE security_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'login_success', 'login_failed', 'password_changed', 'email_changed',
        'account_locked', 'account_unlocked', 'password_reset_requested',
        'password_reset_completed', 'email_verification_sent', 'email_verified',
        'suspicious_activity', '2fa_enabled', '2fa_disabled', 'token_refresh'
    )),
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Account lockout tracking
CREATE TABLE account_lockouts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_failed_attempt TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(user_id)
);

-- Two-Factor Authentication
CREATE TABLE user_2fa (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    secret VARCHAR(255) NOT NULL,
    backup_codes TEXT[], -- Array of backup codes
    enabled BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Session management for refresh tokens
CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    ip_address INET,
    user_agent TEXT,
    device_info JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Add email verification status to users table
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

-- Indexes for performance
CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verification_tokens_user_type ON email_verification_tokens(user_id, token_type);
CREATE INDEX idx_email_verification_tokens_expires ON email_verification_tokens(expires_at) WHERE used_at IS NULL;

CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at) WHERE used_at IS NULL;

CREATE INDEX idx_security_events_user_created ON security_events(user_id, created_at DESC);
CREATE INDEX idx_security_events_type_created ON security_events(event_type, created_at DESC);
CREATE INDEX idx_security_events_ip ON security_events(ip_address);

CREATE INDEX idx_account_lockouts_user ON account_lockouts(user_id);
CREATE INDEX idx_account_lockouts_locked_until ON account_lockouts(locked_until) WHERE locked_until IS NOT NULL;

CREATE INDEX idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at) WHERE is_active = true;

-- Function to clean up expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    -- Clean up expired email verification tokens
    DELETE FROM email_verification_tokens 
    WHERE expires_at < now() AND used_at IS NULL;
    
    -- Clean up expired password reset tokens
    DELETE FROM password_reset_tokens 
    WHERE expires_at < now() AND used_at IS NULL;
    
    -- Clean up expired sessions
    UPDATE user_sessions 
    SET is_active = false 
    WHERE expires_at < now() AND is_active = true;
    
    -- Clean up old inactive sessions (older than 30 days)
    DELETE FROM user_sessions 
    WHERE is_active = false AND last_used_at < now() - INTERVAL '30 days';
    
    -- Clean up old security events (older than 1 year)
    DELETE FROM security_events 
    WHERE created_at < now() - INTERVAL '1 year';
END;
$$ LANGUAGE plpgsql;

-- Function to automatically unlock accounts after lockout period
CREATE OR REPLACE FUNCTION unlock_expired_accounts()
RETURNS void AS $$
BEGIN
    UPDATE account_lockouts 
    SET locked_until = NULL, failed_attempts = 0
    WHERE locked_until IS NOT NULL AND locked_until < now();
END;
$$ LANGUAGE plpgsql;

-- Function to track security events
CREATE OR REPLACE FUNCTION log_security_event(
    p_user_id BIGINT,
    p_event_type VARCHAR(50),
    p_description TEXT DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT NULL
)
RETURNS void AS $$
BEGIN
    INSERT INTO security_events (user_id, event_type, description, ip_address, user_agent, metadata)
    VALUES (p_user_id, p_event_type, p_description, p_ip_address, p_user_agent, p_metadata);
END;
$$ LANGUAGE plpgsql;
