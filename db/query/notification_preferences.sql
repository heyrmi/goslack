-- name: CreateNotificationPreference :one
INSERT INTO notification_preferences (
    user_id, workspace_id, channel_id, notification_type,
    email_notifications, push_notifications, desktop_notifications,
    keywords, do_not_disturb_start, do_not_disturb_end, timezone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetNotificationPreference :one
SELECT * FROM notification_preferences
WHERE user_id = $1 AND workspace_id = $2 AND channel_id = $3;

-- name: GetUserNotificationPreferences :many
SELECT * FROM notification_preferences
WHERE user_id = $1 AND workspace_id = $2
ORDER BY channel_id NULLS FIRST;

-- name: GetGlobalNotificationPreference :one
SELECT * FROM notification_preferences
WHERE user_id = $1 AND workspace_id = $2 AND channel_id IS NULL;

-- name: UpdateNotificationPreference :one
UPDATE notification_preferences
SET 
    notification_type = $4,
    email_notifications = $5,
    push_notifications = $6,
    desktop_notifications = $7,
    keywords = $8,
    do_not_disturb_start = $9,
    do_not_disturb_end = $10,
    timezone = $11,
    updated_at = now()
WHERE user_id = $1 AND workspace_id = $2 AND channel_id = $3
RETURNING *;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (
    user_id, workspace_id, channel_id, notification_type,
    email_notifications, push_notifications, desktop_notifications,
    keywords, do_not_disturb_start, do_not_disturb_end, timezone
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (user_id, workspace_id, channel_id)
DO UPDATE SET
    notification_type = EXCLUDED.notification_type,
    email_notifications = EXCLUDED.email_notifications,
    push_notifications = EXCLUDED.push_notifications,
    desktop_notifications = EXCLUDED.desktop_notifications,
    keywords = EXCLUDED.keywords,
    do_not_disturb_start = EXCLUDED.do_not_disturb_start,
    do_not_disturb_end = EXCLUDED.do_not_disturb_end,
    timezone = EXCLUDED.timezone,
    updated_at = now()
RETURNING *;

-- name: DeleteNotificationPreference :exec
DELETE FROM notification_preferences
WHERE user_id = $1 AND workspace_id = $2 AND channel_id = $3;

-- name: DeleteUserNotificationPreferences :exec
DELETE FROM notification_preferences
WHERE user_id = $1;

-- name: IsInDoNotDisturbMode :one
SELECT CASE 
    WHEN do_not_disturb_start IS NULL OR do_not_disturb_end IS NULL THEN false
    WHEN do_not_disturb_start <= do_not_disturb_end THEN
        CURRENT_TIME BETWEEN do_not_disturb_start AND do_not_disturb_end
    ELSE
        CURRENT_TIME >= do_not_disturb_start OR CURRENT_TIME <= do_not_disturb_end
END as is_in_dnd_mode
FROM notification_preferences
WHERE user_id = $1 AND workspace_id = $2 AND channel_id IS NULL;
