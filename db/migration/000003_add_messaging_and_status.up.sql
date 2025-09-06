-- Core messaging table
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE, -- NULL for direct messages
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for channel messages
    content TEXT NOT NULL CHECK (LENGTH(content) <= 4000),
    message_type VARCHAR(20) NOT NULL CHECK (message_type IN ('channel', 'direct')),
    thread_id BIGINT REFERENCES messages(id) ON DELETE CASCADE, -- For future threading
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ, -- Soft delete support
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Online/offline status tracking
CREATE TABLE user_status (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'away', 'busy', 'offline')),
    custom_status TEXT CHECK (LENGTH(custom_status) <= 100),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Track channel membership for private channels
CREATE TABLE channel_members (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(channel_id, user_id)
);

-- Messaging indexes
CREATE INDEX idx_messages_channel_created_at ON messages (channel_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_direct_users ON messages (workspace_id, sender_id, receiver_id, created_at DESC) WHERE message_type = 'direct' AND deleted_at IS NULL;
CREATE INDEX idx_messages_workspace_created_at ON messages (workspace_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_thread_id ON messages (thread_id) WHERE thread_id IS NOT NULL;

-- Status indexes
CREATE INDEX idx_user_status_workspace ON user_status (workspace_id, status);
CREATE INDEX idx_user_status_activity ON user_status (last_activity_at);

-- Channel members indexes
CREATE INDEX idx_channel_members_channel ON channel_members (channel_id);
CREATE INDEX idx_channel_members_user ON channel_members (user_id);

-- Add constraint to ensure message type consistency
CREATE OR REPLACE FUNCTION check_message_type_consistency()
RETURNS TRIGGER AS $$
BEGIN
    -- For channel messages, channel_id must be set and receiver_id must be NULL
    IF NEW.message_type = 'channel' THEN
        IF NEW.channel_id IS NULL OR NEW.receiver_id IS NOT NULL THEN
            RAISE EXCEPTION 'Channel messages must have channel_id set and receiver_id NULL';
        END IF;
    END IF;
    
    -- For direct messages, receiver_id must be set and channel_id must be NULL
    IF NEW.message_type = 'direct' THEN
        IF NEW.receiver_id IS NULL OR NEW.channel_id IS NOT NULL THEN
            RAISE EXCEPTION 'Direct messages must have receiver_id set and channel_id NULL';
        END IF;
        
        -- Ensure sender and receiver are different
        IF NEW.sender_id = NEW.receiver_id THEN
            RAISE EXCEPTION 'Sender and receiver cannot be the same user';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce message type consistency
CREATE TRIGGER trigger_check_message_type_consistency
    BEFORE INSERT OR UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION check_message_type_consistency();

-- Add constraint to ensure user status belongs to correct workspace
CREATE OR REPLACE FUNCTION check_user_status_workspace()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if the user belongs to the specified workspace
    IF NOT EXISTS (
        SELECT 1 
        FROM users u 
        WHERE u.id = NEW.user_id 
        AND u.workspace_id = NEW.workspace_id
    ) THEN
        RAISE EXCEPTION 'User status must be for a workspace the user belongs to';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce user status workspace constraint
CREATE TRIGGER trigger_check_user_status_workspace
    BEFORE INSERT OR UPDATE ON user_status
    FOR EACH ROW
    EXECUTE FUNCTION check_user_status_workspace();