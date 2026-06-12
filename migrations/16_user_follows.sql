CREATE TABLE IF NOT EXISTS user_follows (
    follower_id TEXT NOT NULL,
    followed_id TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, followed_id),
    CHECK (follower_id <> followed_id),
    FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (followed_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_follows_followed ON user_follows(followed_id, created_at DESC);
