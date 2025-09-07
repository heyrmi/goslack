-- File storage and management tables

-- Core file storage table
CREATE TABLE files (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    uploader_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_filename VARCHAR(255) NOT NULL,
    stored_filename VARCHAR(255) NOT NULL UNIQUE,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    mime_type VARCHAR(100) NOT NULL,
    file_hash VARCHAR(64) NOT NULL, -- SHA-256 for deduplication
    is_public BOOLEAN NOT NULL DEFAULT false,
    upload_completed BOOLEAN NOT NULL DEFAULT false,
    thumbnail_path TEXT, -- Path to thumbnail if generated
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Junction table linking files to messages
CREATE TABLE message_files (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(message_id, file_id)
);

-- File sharing and permissions (for future enhancement)
CREATE TABLE file_shares (
    id BIGSERIAL PRIMARY KEY,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    shared_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    shared_with_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) NOT NULL DEFAULT 'view' CHECK (permission IN ('view', 'download')),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    -- Either channel_id OR shared_with_user_id must be set, not both
    CHECK ((channel_id IS NOT NULL AND shared_with_user_id IS NULL) OR 
           (channel_id IS NULL AND shared_with_user_id IS NOT NULL))
);

-- Indexes for performance
CREATE INDEX idx_files_workspace ON files (workspace_id, created_at DESC);
CREATE INDEX idx_files_uploader ON files (uploader_id);
CREATE INDEX idx_files_hash ON files (file_hash);
CREATE INDEX idx_files_completed ON files (upload_completed, created_at DESC);
CREATE INDEX idx_message_files_message ON message_files (message_id);
CREATE INDEX idx_message_files_file ON message_files (file_id);
CREATE INDEX idx_file_shares_file ON file_shares (file_id);
CREATE INDEX idx_file_shares_channel ON file_shares (channel_id);
CREATE INDEX idx_file_shares_user ON file_shares (shared_with_user_id);

-- Add new message content types
ALTER TABLE messages 
ADD COLUMN content_type VARCHAR(20) NOT NULL DEFAULT 'text' 
CHECK (content_type IN ('text', 'file', 'image', 'system'));

-- Function to ensure file belongs to correct workspace
CREATE OR REPLACE FUNCTION check_file_workspace_consistency()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if uploader belongs to the specified workspace
    IF NOT EXISTS (
        SELECT 1 
        FROM users u 
        WHERE u.id = NEW.uploader_id 
        AND u.workspace_id = NEW.workspace_id
    ) THEN
        RAISE EXCEPTION 'File uploader must belong to the specified workspace';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce file workspace consistency
CREATE TRIGGER trigger_check_file_workspace_consistency
    BEFORE INSERT OR UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION check_file_workspace_consistency();

-- Function to ensure message-file relationship is valid
CREATE OR REPLACE FUNCTION check_message_file_consistency()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if message and file belong to the same workspace
    IF NOT EXISTS (
        SELECT 1 
        FROM messages m
        JOIN files f ON f.workspace_id = m.workspace_id
        WHERE m.id = NEW.message_id 
        AND f.id = NEW.file_id
    ) THEN
        RAISE EXCEPTION 'Message and file must belong to the same workspace';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce message-file consistency
CREATE TRIGGER trigger_check_message_file_consistency
    BEFORE INSERT OR UPDATE ON message_files
    FOR EACH ROW
    EXECUTE FUNCTION check_message_file_consistency();
