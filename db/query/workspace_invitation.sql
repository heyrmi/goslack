-- name: CreateWorkspaceInvitation :one
INSERT INTO workspace_invitations (
    workspace_id,
    inviter_id,
    invitee_email,
    invitee_id,
    invitation_code,
    role,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetWorkspaceInvitation :one
SELECT * FROM workspace_invitations
WHERE id = $1 LIMIT 1;

-- name: GetWorkspaceInvitationByCode :one
SELECT * FROM workspace_invitations
WHERE invitation_code = $1 AND status = 'pending' AND expires_at > NOW()
LIMIT 1;

-- name: ListWorkspaceInvitations :many
SELECT * FROM workspace_invitations
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: AcceptWorkspaceInvitation :one
UPDATE workspace_invitations
SET 
    status = 'accepted',
    accepted_at = NOW(),
    invitee_id = $2
WHERE invitation_code = $1 AND status = 'pending' AND expires_at > NOW()
RETURNING *;

-- name: DeclineWorkspaceInvitation :one
UPDATE workspace_invitations
SET status = 'declined'
WHERE invitation_code = $1 AND status = 'pending'
RETURNING *;

-- name: ExpireWorkspaceInvitation :exec
UPDATE workspace_invitations
SET status = 'expired'
WHERE id = $1;

-- name: DeleteWorkspaceInvitation :exec
DELETE FROM workspace_invitations
WHERE id = $1;

-- name: GetPendingInvitationsForUser :many
SELECT wi.*, w.name as workspace_name, u.first_name as inviter_first_name, u.last_name as inviter_last_name
FROM workspace_invitations wi
JOIN workspaces w ON wi.workspace_id = w.id
JOIN users u ON wi.inviter_id = u.id
WHERE wi.invitee_email = $1 AND wi.status = 'pending' AND wi.expires_at > NOW()
ORDER BY wi.created_at DESC;
