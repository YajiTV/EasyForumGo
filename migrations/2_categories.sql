CREATE TABLE IF NOT EXISTS categories (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT
);
