-- Drop triggers
DROP TRIGGER IF EXISTS trigger_check_channel_creator_workspace ON channels;
DROP TRIGGER IF EXISTS trigger_check_user_workspace_organization ON users;

-- Drop functions
DROP FUNCTION IF EXISTS check_channel_creator_workspace();
DROP FUNCTION IF EXISTS check_user_workspace_organization();

-- Drop indexes (unique indexes will be dropped automatically with columns)
DROP INDEX IF EXISTS idx_workspaces_organization_id;
DROP INDEX IF EXISTS idx_channels_workspace_id;
DROP INDEX IF EXISTS idx_channels_created_by;
DROP INDEX IF EXISTS idx_users_workspace_id;

-- Remove columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS role,
DROP COLUMN IF EXISTS workspace_id;

-- Drop tables (in reverse order of creation due to foreign keys)
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS workspaces;
