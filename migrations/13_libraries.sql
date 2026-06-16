CREATE TABLE IF NOT EXISTS libraries (
    id         TEXT     PRIMARY KEY,
    user_id    TEXT     NOT NULL,
    name       TEXT     NOT NULL COLLATE NOCASE CHECK(length(trim(name)) BETWEEN 1 AND 60),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    UNIQUE (user_id, name)
);
