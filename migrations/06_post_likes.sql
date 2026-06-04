CREATE TABLE IF NOT EXISTS post_likes (
    id         TEXT     PRIMARY KEY,
    post_id    TEXT     NOT NULL,
    user_id    TEXT     NOT NULL,
    is_like    INTEGER  NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (post_id) REFERENCES posts(id)  ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)  ON DELETE CASCADE,

    UNIQUE (post_id, user_id)
);