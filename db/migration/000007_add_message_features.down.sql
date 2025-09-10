-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_unread_count ON messages;
DROP TRIGGER IF EXISTS trigger_message_search_index_delete ON messages;
DROP TRIGGER IF EXISTS trigger_message_search_index_update ON messages;
DROP TRIGGER IF EXISTS trigger_message_search_index_insert ON messages;
DROP TRIGGER IF EXISTS trigger_update_thread_reply_count_delete ON messages;
DROP TRIGGER IF EXISTS trigger_update_thread_reply_count_insert ON messages;

-- Drop functions
DROP FUNCTION IF EXISTS update_unread_count();
DROP FUNCTION IF EXISTS update_message_search_index();
DROP FUNCTION IF EXISTS update_thread_reply_count();

-- Drop indexes
DROP INDEX IF EXISTS idx_message_search_content;
DROP INDEX IF EXISTS idx_message_search_user;
DROP INDEX IF EXISTS idx_message_search_channel;
DROP INDEX IF EXISTS idx_message_search_workspace;

DROP INDEX IF EXISTS idx_notification_preferences_channel;
DROP INDEX IF EXISTS idx_notification_preferences_user_workspace;

DROP INDEX IF EXISTS idx_scheduled_messages_scheduled_for;
DROP INDEX IF EXISTS idx_scheduled_messages_user;

DROP INDEX IF EXISTS idx_message_drafts_updated;
DROP INDEX IF EXISTS idx_message_drafts_channel;
DROP INDEX IF EXISTS idx_message_drafts_user;

DROP INDEX IF EXISTS idx_pinned_messages_pinned_by;
DROP INDEX IF EXISTS idx_pinned_messages_channel;

DROP INDEX IF EXISTS idx_unread_messages_unread_count;
DROP INDEX IF EXISTS idx_unread_messages_channel;
DROP INDEX IF EXISTS idx_unread_messages_user_workspace;

DROP INDEX IF EXISTS idx_message_mentions_type;
DROP INDEX IF EXISTS idx_message_mentions_user;
DROP INDEX IF EXISTS idx_message_mentions_message;

DROP INDEX IF EXISTS idx_message_reactions_emoji;
DROP INDEX IF EXISTS idx_message_reactions_user;
DROP INDEX IF EXISTS idx_message_reactions_message;

-- Remove columns from messages table
ALTER TABLE messages DROP COLUMN IF EXISTS last_reply_at;
ALTER TABLE messages DROP COLUMN IF EXISTS reply_count;

-- Drop tables
DROP TABLE IF EXISTS message_search_index;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS scheduled_messages;
DROP TABLE IF EXISTS message_drafts;
DROP TABLE IF EXISTS pinned_messages;
DROP TABLE IF EXISTS unread_messages;
DROP TABLE IF EXISTS message_mentions;
DROP TABLE IF EXISTS message_reactions;
