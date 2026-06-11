CREATE TABLE IF NOT EXISTS user_restrictions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK(type IN ('mute', 'ban')),
    reason     TEXT NOT NULL,
    expires_at DATETIME,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
