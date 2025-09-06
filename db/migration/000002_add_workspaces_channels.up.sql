-- Create workspaces table
CREATE TABLE workspaces (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Create channels table
CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    is_private BOOLEAN NOT NULL DEFAULT false,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Add workspace_id and role columns to users table
ALTER TABLE users 
ADD COLUMN workspace_id BIGINT REFERENCES workspaces(id) ON DELETE SET NULL,
ADD COLUMN role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member'));

-- Create function to ensure user's workspace belongs to same organization
CREATE OR REPLACE FUNCTION check_user_workspace_organization()
RETURNS TRIGGER AS $$
BEGIN
    -- If workspace_id is NULL, skip the check
    IF NEW.workspace_id IS NULL THEN
        RETURN NEW;
    END IF;
    
    -- Check if the workspace belongs to the same organization as the user
    IF NOT EXISTS (
        SELECT 1 
        FROM workspaces w 
        WHERE w.id = NEW.workspace_id 
        AND w.organization_id = NEW.organization_id
    ) THEN
        RAISE EXCEPTION 'User workspace must belong to the same organization';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce the constraint
CREATE TRIGGER trigger_check_user_workspace_organization
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION check_user_workspace_organization();

-- Create function to ensure channel creator is member of workspace
CREATE OR REPLACE FUNCTION check_channel_creator_workspace()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if the creator is a member of the workspace
    IF NOT EXISTS (
        SELECT 1 
        FROM users u 
        WHERE u.id = NEW.created_by 
        AND u.workspace_id = NEW.workspace_id
    ) THEN
        RAISE EXCEPTION 'Channel creator must be a member of the workspace';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce the constraint
CREATE TRIGGER trigger_check_channel_creator_workspace
    BEFORE INSERT OR UPDATE ON channels
    FOR EACH ROW
    EXECUTE FUNCTION check_channel_creator_workspace();

-- Create indexes for better performance
CREATE INDEX ON workspaces (organization_id);
CREATE INDEX ON channels (workspace_id);
CREATE INDEX ON channels (created_by);
CREATE INDEX ON users (workspace_id);
CREATE UNIQUE INDEX ON workspaces (organization_id, name);
CREATE UNIQUE INDEX ON channels (workspace_id, name);
