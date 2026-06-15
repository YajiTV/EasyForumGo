CREATE TABLE IF NOT EXISTS flagged_keywords (
    id         TEXT PRIMARY KEY,
    word       TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO flagged_keywords (id, word) VALUES
    ('kw-001', 'insulte'),
    ('kw-002', 'spam'),
    ('kw-003', 'haine'),
    ('kw-004', 'obscene'),
    ('kw-005', 'illegal');
