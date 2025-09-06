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
