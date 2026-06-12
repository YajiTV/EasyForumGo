CREATE TABLE IF NOT EXISTS reports (
    id          TEXT PRIMARY KEY,
    reporter_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK(target_type IN ('post', 'comment', 'user')),
    target_id   TEXT NOT NULL,
    category    TEXT NOT NULL CHECK(category IN ('spam', 'harassment', 'inappropriate_content', 'hate_speech', 'other')),
    reason      TEXT,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'resolved', 'dismissed')),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(reporter_id, target_type, target_id)
);
