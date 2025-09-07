-- Rollback file management tables and changes

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_check_message_file_consistency ON message_files;
DROP TRIGGER IF EXISTS trigger_check_file_workspace_consistency ON files;

-- Drop functions
DROP FUNCTION IF EXISTS check_message_file_consistency();
DROP FUNCTION IF EXISTS check_file_workspace_consistency();

-- Remove content_type column from messages
ALTER TABLE messages DROP COLUMN IF EXISTS content_type;

-- Drop tables in reverse order
DROP TABLE IF EXISTS file_shares;
DROP TABLE IF EXISTS message_files;
DROP TABLE IF EXISTS files;
