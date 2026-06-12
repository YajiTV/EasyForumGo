CREATE TABLE IF NOT EXISTS library_posts (
    library_id TEXT     NOT NULL,
    post_id    TEXT     NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,

    PRIMARY KEY (library_id, post_id)
);
