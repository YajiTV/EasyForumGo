CREATE TABLE IF NOT EXISTS notifications (
    id         TEXT     PRIMARY KEY,
    user_id    TEXT     NOT NULL,
    actor_id   TEXT     NOT NULL,
    type       TEXT     NOT NULL,
    post_id    TEXT,
    comment_id TEXT,
    is_read    INTEGER  NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE CASCADE,
    FOREIGN KEY (actor_id)   REFERENCES users(id)    ON DELETE CASCADE,
    FOREIGN KEY (post_id)    REFERENCES posts(id)    ON DELETE CASCADE,
    FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
);
