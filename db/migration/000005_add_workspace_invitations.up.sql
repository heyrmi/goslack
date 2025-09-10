-- Create workspace_invitations table
CREATE TABLE workspace_invitations (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_email VARCHAR(255) NOT NULL,
    invitee_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    invitation_code VARCHAR(255) UNIQUE NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Create indexes for better performance
CREATE INDEX ON workspace_invitations (workspace_id);
CREATE INDEX ON workspace_invitations (inviter_id);
CREATE INDEX ON workspace_invitations (invitee_email);
CREATE INDEX ON workspace_invitations (invitation_code);
CREATE INDEX ON workspace_invitations (status);
CREATE INDEX ON workspace_invitations (expires_at);

-- Create function to generate invitation codes
CREATE OR REPLACE FUNCTION generate_invitation_code() RETURNS TEXT AS $$
BEGIN
    RETURN UPPER(SUBSTRING(MD5(RANDOM()::TEXT || NOW()::TEXT), 1, 8));
END;
$$ LANGUAGE plpgsql;

-- Create function to auto-expire invitations
CREATE OR REPLACE FUNCTION expire_old_invitations() RETURNS VOID AS $$
BEGIN
    UPDATE workspace_invitations 
    SET status = 'expired' 
    WHERE status = 'pending' AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;
