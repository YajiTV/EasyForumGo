CREATE TABLE IF NOT EXISTS comment_likes (
    id         TEXT     PRIMARY KEY,
    comment_id TEXT     NOT NULL,
    user_id    TEXT     NOT NULL,
    is_like    INTEGER  NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE CASCADE,

    UNIQUE (comment_id, user_id)
);
