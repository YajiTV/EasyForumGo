CREATE TABLE IF NOT EXISTS moderation_actions (
    id             TEXT PRIMARY KEY,
    moderator_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_id      TEXT REFERENCES reports(id) ON DELETE SET NULL,
    action_type    TEXT NOT NULL CHECK(action_type IN ('delete_post', 'delete_comment', 'warn_user', 'mute_user', 'ban_user', 'change_role')),
    target_id      TEXT NOT NULL,
    reason         TEXT,
    duration_hours INTEGER,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
