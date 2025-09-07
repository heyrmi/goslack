-- name: CreateFile :one
INSERT INTO files (
    workspace_id, uploader_id, original_filename, stored_filename, 
    file_path, file_size, mime_type, file_hash, is_public, 
    upload_completed, thumbnail_path
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetFile :one
SELECT * FROM files
WHERE id = $1 LIMIT 1;

-- name: GetFileByHash :one
SELECT * FROM files
WHERE file_hash = $1 AND workspace_id = $2 AND upload_completed = true
LIMIT 1;

-- name: UpdateFileUploadStatus :exec
UPDATE files
SET upload_completed = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateFileThumbnail :exec
UPDATE files
SET thumbnail_path = $2, updated_at = now()
WHERE id = $1;

-- name: ListWorkspaceFiles :many
SELECT f.*, u.first_name as uploader_first_name, u.last_name as uploader_last_name, u.email as uploader_email
FROM files f
JOIN users u ON f.uploader_id = u.id
WHERE f.workspace_id = $1 AND f.upload_completed = true
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUserFiles :many
SELECT f.*, u.first_name as uploader_first_name, u.last_name as uploader_last_name, u.email as uploader_email
FROM files f
JOIN users u ON f.uploader_id = u.id
WHERE f.uploader_id = $1 AND f.workspace_id = $2 AND f.upload_completed = true
ORDER BY f.created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteFile :exec
DELETE FROM files
WHERE id = $1 AND uploader_id = $2;

-- name: GetFileWithPermissionCheck :one
SELECT f.*, u.first_name as uploader_first_name, u.last_name as uploader_last_name, u.email as uploader_email
FROM files f
JOIN users u ON f.uploader_id = u.id
WHERE f.id = $1 AND f.workspace_id = $2 AND f.upload_completed = true
LIMIT 1;

-- name: CreateMessageFile :one
INSERT INTO message_files (message_id, file_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetMessageFiles :many
SELECT f.*, u.first_name as uploader_first_name, u.last_name as uploader_last_name, u.email as uploader_email
FROM message_files mf
JOIN files f ON mf.file_id = f.id
JOIN users u ON f.uploader_id = u.id
WHERE mf.message_id = $1
ORDER BY mf.created_at ASC;

-- name: GetFileMessages :many
SELECT m.*, u.first_name as sender_first_name, u.last_name as sender_last_name, u.email as sender_email
FROM message_files mf
JOIN messages m ON mf.message_id = m.id
JOIN users u ON m.sender_id = u.id
WHERE mf.file_id = $1 AND m.deleted_at IS NULL
ORDER BY m.created_at DESC;

-- name: DeleteMessageFile :exec
DELETE FROM message_files
WHERE message_id = $1 AND file_id = $2;

-- name: CreateFileShare :one
INSERT INTO file_shares (file_id, shared_by, channel_id, shared_with_user_id, permission, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFileShares :many
SELECT fs.*, u.first_name as shared_by_first_name, u.last_name as shared_by_last_name, u.email as shared_by_email
FROM file_shares fs
JOIN users u ON fs.shared_by = u.id
WHERE fs.file_id = $1
ORDER BY fs.created_at DESC;

-- name: CheckFileAccess :one
-- Check if user has access to file through direct ownership, channel membership, or direct share
SELECT CASE 
    WHEN f.uploader_id = $2 THEN true
    WHEN f.is_public = true THEN true
    WHEN EXISTS (
        SELECT 1 FROM file_shares fs 
        WHERE fs.file_id = $1 AND fs.shared_with_user_id = $2 
        AND (fs.expires_at IS NULL OR fs.expires_at > now())
    ) THEN true
    WHEN EXISTS (
        SELECT 1 FROM file_shares fs 
        JOIN channel_members cm ON fs.channel_id = cm.channel_id
        WHERE fs.file_id = $1 AND cm.user_id = $2
        AND (fs.expires_at IS NULL OR fs.expires_at > now())
    ) THEN true
    ELSE false
END as has_access
FROM files f
WHERE f.id = $1;

-- name: GetFileStats :one
SELECT 
    COUNT(*) as total_files,
    COALESCE(SUM(file_size), 0) as total_size,
    COUNT(*) FILTER (WHERE mime_type LIKE 'image/%') as image_count,
    COUNT(*) FILTER (WHERE mime_type = 'application/pdf') as pdf_count
FROM files 
WHERE workspace_id = $1 AND upload_completed = true;

-- name: CleanupIncompleteUploads :exec
DELETE FROM files 
WHERE upload_completed = false 
AND created_at < now() - INTERVAL '1 hour';

-- name: GetDuplicateFiles :many
SELECT file_hash, COUNT(*) as count, ARRAY_AGG(id) as file_ids, SUM(file_size) as total_size
FROM files 
WHERE workspace_id = $1 AND upload_completed = true
GROUP BY file_hash 
HAVING COUNT(*) > 1
ORDER BY count DESC;
