ALTER TABLE posts ADD COLUMN approved INTEGER NOT NULL DEFAULT 0;
-- Approve all existing posts so pre-existing content remains visible
UPDATE posts SET approved = 1;
