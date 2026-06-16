CREATE TABLE IF NOT EXISTS content_reviews (
    id           TEXT PRIMARY KEY,
    content_type TEXT NOT NULL CHECK(content_type IN ('post', 'comment')),
    content_id   TEXT NOT NULL,
    reviewer_id  TEXT REFERENCES users(id) ON DELETE SET NULL,
    decision     TEXT NOT NULL CHECK(decision IN ('approved', 'rejected')),
    reason       TEXT,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
