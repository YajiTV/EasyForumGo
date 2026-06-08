package repository

import (
	"path/filepath"
	"testing"
)

// TestInitDB verifies migrations and foreign key cascades
func TestInitDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forum.db")
	migrationsDir := filepath.Join("..", "..", "migrations")

	db, err := InitDB(dbPath, migrationsDir)
	if err != nil {
		t.Fatalf("InitDB returned an error: %v", err)
	}
	defer db.Close()

	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 10 {
		t.Fatalf("expected 10 migrations, got %d", migrationCount)
	}

	var foreignKeysEnabled int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeysEnabled); err != nil {
		t.Fatalf("read foreign_keys pragma: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Fatalf("expected foreign keys to be enabled, got %d", foreignKeysEnabled)
	}

	if _, err := db.Exec(`INSERT INTO users (id, email, username, password) VALUES (?, ?, ?, ?)`,
		"user-id", "user@example.test", "test-user", "hash"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, user_id, title, content) VALUES (?, ?, ?, ?)`,
		"post-id", "user-id", "Test post", "Test content"); err != nil {
		t.Fatalf("insert post: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, "user-id"); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var postCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE id = ?`, "post-id").Scan(&postCount); err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if postCount != 0 {
		t.Fatalf("expected cascade to delete the post, got %d posts", postCount)
	}
}
