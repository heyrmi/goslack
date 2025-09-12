-- Message threading, reactions, mentions, and advanced features

-- Add content_type column to messages table (if not exists)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content_type VARCHAR(20) NOT NULL DEFAULT 'text' 
CHECK (content_type IN ('text', 'file', 'image', 'system'));

-- Message reactions
CREATE TABLE message_reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(message_id, user_id, emoji)
);

-- Message mentions
CREATE TABLE message_mentions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    mentioned_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for @channel, @here
    mention_type VARCHAR(20) NOT NULL CHECK (mention_type IN ('user', 'channel', 'here', 'everyone')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Unread messages tracking
CREATE TABLE unread_messages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE, -- NULL for direct messages
    last_read_message_id BIGINT REFERENCES messages(id) ON DELETE CASCADE,
    unread_count INTEGER NOT NULL DEFAULT 0,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(user_id, channel_id) -- One record per user per channel/DM
);

-- Message pinning
CREATE TABLE pinned_messages (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    pinned_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pinned_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(message_id) -- A message can only be pinned once
);

-- Message drafts
CREATE TABLE message_drafts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE, -- NULL for direct messages
    receiver_id BIGINT REFERENCES users(id) ON DELETE CASCADE, -- NULL for channel messages
    thread_id BIGINT REFERENCES messages(id) ON DELETE CASCADE, -- NULL for non-thread messages
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(user_id, channel_id, receiver_id, thread_id) -- One draft per conversation/thread
);

-- Scheduled messages
CREATE TABLE scheduled_messages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    receiver_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    thread_id BIGINT REFERENCES messages(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    content_type VARCHAR(20) NOT NULL DEFAULT 'text',
    scheduled_for TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Notification preferences
CREATE TABLE notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE, -- NULL for global preferences
    notification_type VARCHAR(30) NOT NULL CHECK (notification_type IN (
        'all_messages', 'mentions_only', 'nothing', 'direct_messages', 'keywords'
    )),
    email_notifications BOOLEAN NOT NULL DEFAULT true,
    push_notifications BOOLEAN NOT NULL DEFAULT true,
    desktop_notifications BOOLEAN NOT NULL DEFAULT true,
    keywords TEXT[], -- Array of keywords to notify on
    do_not_disturb_start TIME,
    do_not_disturb_end TIME,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT (now()),
    UNIQUE(user_id, workspace_id, channel_id)
);

-- Message search index (for full-text search)
CREATE TABLE message_search_index (
    message_id BIGINT PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_vector tsvector NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT (now())
);

-- Indexes for performance
CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);
CREATE INDEX idx_message_reactions_user ON message_reactions(user_id);
CREATE INDEX idx_message_reactions_emoji ON message_reactions(emoji);

CREATE INDEX idx_message_mentions_message ON message_mentions(message_id);
CREATE INDEX idx_message_mentions_user ON message_mentions(mentioned_user_id);
CREATE INDEX idx_message_mentions_type ON message_mentions(mention_type);

CREATE INDEX idx_unread_messages_user_workspace ON unread_messages(user_id, workspace_id);
CREATE INDEX idx_unread_messages_channel ON unread_messages(channel_id);
CREATE INDEX idx_unread_messages_unread_count ON unread_messages(unread_count) WHERE unread_count > 0;

CREATE INDEX idx_pinned_messages_channel ON pinned_messages(channel_id);
CREATE INDEX idx_pinned_messages_pinned_by ON pinned_messages(pinned_by);

CREATE INDEX idx_message_drafts_user ON message_drafts(user_id);
CREATE INDEX idx_message_drafts_channel ON message_drafts(channel_id);
CREATE INDEX idx_message_drafts_updated ON message_drafts(updated_at);

CREATE INDEX idx_scheduled_messages_user ON scheduled_messages(user_id);
CREATE INDEX idx_scheduled_messages_scheduled_for ON scheduled_messages(scheduled_for) WHERE sent_at IS NULL AND cancelled_at IS NULL;

CREATE INDEX idx_notification_preferences_user_workspace ON notification_preferences(user_id, workspace_id);
CREATE INDEX idx_notification_preferences_channel ON notification_preferences(channel_id);

CREATE INDEX idx_message_search_workspace ON message_search_index(workspace_id);
CREATE INDEX idx_message_search_channel ON message_search_index(channel_id);
CREATE INDEX idx_message_search_user ON message_search_index(user_id);
CREATE INDEX idx_message_search_content ON message_search_index USING gin(content_vector);

-- Add thread reply count to messages table
ALTER TABLE messages ADD COLUMN reply_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN last_reply_at TIMESTAMPTZ;

-- Update thread reply counts when messages are added/deleted
CREATE OR REPLACE FUNCTION update_thread_reply_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Increment reply count for parent message
        IF NEW.thread_id IS NOT NULL THEN
            UPDATE messages 
            SET reply_count = reply_count + 1, 
                last_reply_at = NEW.created_at
            WHERE id = NEW.thread_id;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        -- Decrement reply count for parent message
        IF OLD.thread_id IS NOT NULL THEN
            UPDATE messages 
            SET reply_count = GREATEST(reply_count - 1, 0),
                last_reply_at = (
                    SELECT MAX(created_at) 
                    FROM messages 
                    WHERE thread_id = OLD.thread_id AND deleted_at IS NULL
                )
            WHERE id = OLD.thread_id;
        END IF;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for thread reply count
CREATE TRIGGER trigger_update_thread_reply_count_insert
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_thread_reply_count();

CREATE TRIGGER trigger_update_thread_reply_count_delete
    AFTER UPDATE OF deleted_at ON messages
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
    EXECUTE FUNCTION update_thread_reply_count();

-- Function to update message search index
CREATE OR REPLACE FUNCTION update_message_search_index()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO message_search_index (message_id, workspace_id, channel_id, user_id, content_vector)
        VALUES (
            NEW.id, 
            NEW.workspace_id, 
            NEW.channel_id, 
            NEW.sender_id,
            to_tsvector('english', NEW.content)
        );
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE message_search_index 
        SET content_vector = to_tsvector('english', NEW.content)
        WHERE message_id = NEW.id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        DELETE FROM message_search_index WHERE message_id = OLD.id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for search index
CREATE TRIGGER trigger_message_search_index_insert
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_message_search_index();

CREATE TRIGGER trigger_message_search_index_update
    AFTER UPDATE OF content ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_message_search_index();

CREATE TRIGGER trigger_message_search_index_delete
    AFTER DELETE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_message_search_index();

-- Function to update unread counts
CREATE OR REPLACE FUNCTION update_unread_count()
RETURNS TRIGGER AS $$
BEGIN
    -- Only update for regular messages (not system messages)
    IF NEW.content_type = 'text' OR NEW.content_type = 'file' OR NEW.content_type = 'image' THEN
        -- Update unread count for channel messages
        IF NEW.channel_id IS NOT NULL THEN
            -- Get all channel members except the sender
            INSERT INTO unread_messages (user_id, workspace_id, channel_id, unread_count, updated_at)
            SELECT 
                cm.user_id,
                NEW.workspace_id,
                NEW.channel_id,
                1,
                NEW.created_at
            FROM channel_members cm
            WHERE cm.channel_id = NEW.channel_id 
            AND cm.user_id != NEW.sender_id
            ON CONFLICT (user_id, channel_id)
            DO UPDATE SET 
                unread_count = unread_messages.unread_count + 1,
                updated_at = NEW.created_at;
        END IF;
        
        -- Update unread count for direct messages
        IF NEW.receiver_id IS NOT NULL THEN
            INSERT INTO unread_messages (user_id, workspace_id, channel_id, unread_count, updated_at)
            VALUES (NEW.receiver_id, NEW.workspace_id, NULL, 1, NEW.created_at)
            ON CONFLICT (user_id, channel_id)
            DO UPDATE SET 
                unread_count = unread_messages.unread_count + 1,
                updated_at = NEW.created_at;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for unread count
CREATE TRIGGER trigger_update_unread_count
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_unread_count();
