DELETE FROM notifications
WHERE rowid NOT IN (
    SELECT MIN(rowid)
    FROM notifications
    GROUP BY user_id, actor_id, type, COALESCE(post_id, ''), COALESCE(comment_id, '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_unique_event
ON notifications(user_id, actor_id, type, COALESCE(post_id, ''), COALESCE(comment_id, ''));
