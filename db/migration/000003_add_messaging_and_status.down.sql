-- Drop triggers
DROP TRIGGER IF EXISTS trigger_check_user_status_workspace ON user_status;
DROP TRIGGER IF EXISTS trigger_check_message_type_consistency ON messages;

-- Drop functions
DROP FUNCTION IF EXISTS check_user_status_workspace();
DROP FUNCTION IF EXISTS check_message_type_consistency();

-- Drop indexes
DROP INDEX IF EXISTS idx_channel_members_user;
DROP INDEX IF EXISTS idx_channel_members_channel;
DROP INDEX IF EXISTS idx_user_status_activity;
DROP INDEX IF EXISTS idx_user_status_workspace;
DROP INDEX IF EXISTS idx_messages_thread_id;
DROP INDEX IF EXISTS idx_messages_workspace_created_at;
DROP INDEX IF EXISTS idx_messages_direct_users;
DROP INDEX IF EXISTS idx_messages_channel_created_at;

-- Drop tables (in reverse order of creation due to foreign keys)
DROP TABLE IF EXISTS channel_members;
DROP TABLE IF EXISTS user_status;
DROP TABLE IF EXISTS messages;