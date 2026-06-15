CREATE TABLE oauth_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(provider, provider_user_id),
    UNIQUE(user_id, provider)
);

INSERT OR IGNORE INTO oauth_identities (id, user_id, provider, provider_user_id)
SELECT lower(hex(randomblob(16))), id, oauth_provider, oauth_id
FROM users
WHERE oauth_provider IS NOT NULL
  AND oauth_provider <> ''
  AND oauth_id IS NOT NULL
  AND oauth_id <> '';

CREATE TABLE oauth_pending_flows (
    token TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    picture TEXT NOT NULL DEFAULT '',
    suggested_name TEXT NOT NULL DEFAULT '',
    expires_at DATETIME NOT NULL
);
