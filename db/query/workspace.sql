-- name: CreateWorkspace :one
INSERT INTO workspaces (
    organization_id,
    name
) VALUES (
    $1, $2
)
RETURNING *;

-- name: GetWorkspace :one
SELECT * FROM workspaces
WHERE id = $1 LIMIT 1;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1 LIMIT 1;

-- name: ListWorkspacesByOrganization :many
SELECT * FROM workspaces
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

-- name: GetWorkspaceWithUserCount :one
SELECT 
    w.*,
    COUNT(u.id) as user_count
FROM workspaces w
LEFT JOIN users u ON w.id = u.workspace_id
WHERE w.id = $1
GROUP BY w.id, w.organization_id, w.name, w.created_at
LIMIT 1;

-- name: AddUserToWorkspace :one
UPDATE users
SET 
    workspace_id = $2,
    role = $3
WHERE users.id = $1 AND users.organization_id = (
    SELECT workspaces.organization_id FROM workspaces WHERE workspaces.id = $2
)
RETURNING *;

-- name: RemoveUserFromWorkspace :one
UPDATE users
SET 
    workspace_id = NULL,
    role = 'member'
WHERE users.id = $1 AND users.workspace_id = $2
RETURNING *;

-- name: UpdateWorkspaceMemberRole :one
UPDATE users
SET role = $3
WHERE users.id = $1 AND users.workspace_id = $2
RETURNING *;

-- name: ListWorkspaceMembers :many
SELECT u.id, u.organization_id, u.email, u.first_name, u.last_name, u.role, u.created_at, u.workspace_id
FROM users u
WHERE u.workspace_id = $1
ORDER BY u.role DESC, u.created_at ASC
LIMIT $2
OFFSET $3;

-- name: GetWorkspaceMemberCount :one
SELECT COUNT(*) as member_count
FROM users
WHERE workspace_id = $1;

-- name: CheckUserInWorkspace :one
SELECT EXISTS(
    SELECT 1 FROM users 
    WHERE users.id = $1 AND users.workspace_id = $2
) as is_member;
